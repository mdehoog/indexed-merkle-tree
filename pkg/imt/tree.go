package imt

import (
	"errors"
	"math/big"

	"github.com/mdehoog/gnark-indexed-merkle-tree/pkg/db"
)

const indexKeyPrefix = byte(1)
const hashKeyPrefix = byte(2)
const nodeKeyPrefix = byte(3)
const sizeKeyPrefix = byte(4)

var sizeKey = []byte{sizeKeyPrefix}

type HashFn func([]*big.Int) (*big.Int, error)

type Tree struct {
	tx     db.Transaction
	levels uint64
	feLen  uint64
	hash   HashFn
	root   *big.Int
	size   *uint64
}

func NewTree(tx db.Transaction, levels, feLen uint64, hash HashFn) *Tree {
	return &Tree{
		tx:     tx,
		levels: levels,
		feLen:  feLen,
		hash:   hash,
	}
}

func (t *Tree) Root() (*big.Int, error) {
	if t.root == nil {
		rootBytes, err := t.tx.Get(t.hashKey(0, 0))
		if err != nil {
			return nil, err
		}
		t.root = new(big.Int).SetBytes(rootBytes)
	}
	return t.root, nil
}

func (t *Tree) Size() (uint64, error) {
	if t.size == nil {
		s, err := t.tx.Get(sizeKey)
		if err == nil {
			b := new(big.Int).SetBytes(s).Uint64()
			t.size = &b
		} else if errors.Is(err, db.ErrNotFound) {
			t.size = new(uint64)
		} else {
			return 0, err
		}
	}
	return *t.size, nil
}

func (t *Tree) Get(key *big.Int) (*big.Int, error) {
	i, err := t.keyIndex(key)
	if err != nil {
		return nil, err
	}
	n, err := t.node(i)
	if err != nil {
		return nil, err
	}
	return n.Value, nil
}

func (t *Tree) ProveInclusion(key *big.Int) (*Node, []*big.Int, error) {
	i, err := t.keyIndex(key)
	if err != nil {
		return nil, nil, err
	}
	return t.ProveIndex(i)
}

func (t *Tree) ProveExclusion(key *big.Int) (*Node, []*big.Int, error) {
	i, err := t.lowNullifierIndex(key)
	if err != nil {
		return nil, nil, err
	}
	return t.ProveIndex(i)
}

func (t *Tree) ProveIndex(index uint64) (*Node, []*big.Int, error) {
	n, err := t.node(index)
	if err != nil {
		return nil, nil, err
	}

	siblings := make([]*big.Int, t.levels)
	for level := t.levels; level > 0; index /= 2 {
		level--
		siblingIndex := index + 1 - (index%2)*2
		siblingHashBytes, err := t.tx.Get(t.hashKey(siblingIndex, level))
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, nil, err
		}
		siblings[level] = new(big.Int).SetBytes(siblingHashBytes)
	}

	return n, siblings, nil
}

func (t *Tree) Insert(key, value *big.Int) error {
	_, err := t.tx.Get(t.indexKey(key))
	if err == nil {
		return errors.New("key already exists")
	} else if !errors.Is(err, db.ErrNotFound) {
		return err
	}

	lnIndex, err := t.lowNullifierIndex(key)
	if err != nil {
		return err
	}
	ln, err := t.node(lnIndex)
	if err != nil {
		return err
	}

	size, err := t.Size()
	if err != nil {
		return err
	}
	size++

	newNode := &Node{
		NextIndex: ln.NextIndex,
		Key:       key,
		Value:     value,
		NextKey:   ln.NextKey,
	}
	_, err = t.setNode(size, newNode)
	if err != nil {
		return err
	}

	ln.NextIndex = size
	ln.NextKey = key
	_, err = t.setNode(lnIndex, ln)
	if err != nil {
		return err
	}

	err = t.setKeyIndex(key, size)
	if err != nil {
		return err
	}
	return t.setSize(size)
}

func (t *Tree) Update(key, value *big.Int) error {
	i, err := t.keyIndex(key)
	if err != nil {
		return err
	}
	n, err := t.node(i)
	if err != nil {
		return err
	}
	n.Value = value
	_, err = t.setNode(i, n)
	return err
}

func (t *Tree) setSize(s uint64) error {
	t.size = &s
	return t.tx.Set(sizeKey, new(big.Int).SetUint64(s).Bytes())
}

func (t *Tree) keyIndex(key *big.Int) (uint64, error) {
	lt, err := t.tx.Get(t.indexKey(key))
	if err != nil {
		return 0, err
	}
	return new(big.Int).SetBytes(lt).Uint64(), nil
}

func (t *Tree) setKeyIndex(key *big.Int, index uint64) error {
	return t.tx.Set(t.indexKey(key), new(big.Int).SetUint64(index).Bytes())
}

func (t *Tree) lowNullifierIndex(key *big.Int) (uint64, error) {
	_, lt, err := t.tx.GetLT(t.indexKey(key), t.zeroIndexKey())
	if err != nil {
		return 0, err
	}
	return new(big.Int).SetBytes(lt).Uint64(), nil
}

func (t *Tree) node(index uint64) (*Node, error) {
	b, err := t.tx.Get(t.nodeKey(new(big.Int).SetUint64(index)))
	if errors.Is(err, db.ErrNotFound) && index == 0 {
		// initial state
		return &Node{
			NextIndex: 0,
			Key:       big.NewInt(0),
			Value:     big.NewInt(0),
			NextKey:   big.NewInt(0),
		}, nil
	} else if err != nil {
		return nil, err
	}
	n := &Node{}
	err = n.fromBytes(b)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (t *Tree) setNode(index uint64, n *Node) ([]*big.Int, error) {
	err := t.tx.Set(t.nodeKey(new(big.Int).SetUint64(index)), n.bytes())
	if err != nil {
		return nil, err
	}

	siblings := make([]*big.Int, t.levels)
	h, err := n.hash(t.hash)
	if err != nil {
		return nil, err
	}
	for level := t.levels; level > 0; index /= 2 {
		level--
		siblingIndex := index + 1 - (index%2)*2
		siblingHashBytes, err := t.tx.Get(t.hashKey(siblingIndex, level))
		siblings[level] = new(big.Int).SetBytes(siblingHashBytes)
		if err == nil {
			if index%2 == 0 {
				h, err = t.hash([]*big.Int{siblings[level], h})
			} else {
				h, err = t.hash([]*big.Int{h, siblings[level]})
			}
			if err != nil {
				return nil, err
			}
		} else if !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		err = t.tx.Set(t.hashKey(index, level), h.Bytes())
		if err != nil {
			return nil, err
		}
		if level == 0 {
			t.root = h
		}
	}

	return siblings, nil
}

func (t *Tree) indexKey(key *big.Int) []byte {
	b := key.Bytes()
	prefix := make([]byte, 1+int(t.feLen)-len(b))
	prefix[0] = indexKeyPrefix
	return append(prefix, b...)
}

func (t *Tree) zeroIndexKey() []byte {
	b := make([]byte, 1+int(t.feLen))
	b[0] = indexKeyPrefix
	return b
}

func (t *Tree) hashKey(index, level uint64) []byte {
	total := new(big.Int).Lsh(big.NewInt(1), uint(t.levels+1))
	start := new(big.Int).Lsh(big.NewInt(1), uint(level+2))
	total.Sub(total, start)
	total.Add(total, new(big.Int).SetUint64(index))
	return append([]byte{hashKeyPrefix}, total.Bytes()...)
}

func (t *Tree) nodeKey(key *big.Int) []byte {
	return append([]byte{nodeKeyPrefix}, key.Bytes()...)
}
