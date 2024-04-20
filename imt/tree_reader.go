package imt

import (
	"errors"
	"math/big"

	"github.com/mdehoog/indexed-merkle-tree/db"
)

const indexKeyPrefix = byte(1)
const hashKeyPrefix = byte(2)
const nodeKeyPrefix = byte(3)
const sizeKeyPrefix = byte(4)

var sizeKey = []byte{sizeKeyPrefix}

type TreeReader struct {
	reader db.Reader
	levels uint64
	feLen  uint64
	hash   HashFn
}

func NewTreeReader(reader db.Reader, levels, feLen uint64, hash HashFn) *TreeReader {
	return &TreeReader{
		reader: reader,
		levels: levels,
		feLen:  feLen,
		hash:   hash,
	}
}

func (t *TreeReader) Root() (*big.Int, error) {
	rootNodeBytes, err := t.reader.Get(t.hashKey(0, 0))
	if errors.Is(err, db.ErrNotFound) {
		// initial state: hash of empty node
		initialHash, err := t.initialStateNode().hash(t.hash)
		if err != nil {
			return nil, err
		}
		rootNodeBytes = initialHash.Bytes()
	} else if err != nil {
		return nil, err
	}

	// hash the root node with the size to calculate the final tree root
	size, err := t.Size()
	if err != nil {
		return nil, err
	}
	return t.hash([]*big.Int{new(big.Int).SetBytes(rootNodeBytes), new(big.Int).SetUint64(size)})
}

func (t *TreeReader) Size() (uint64, error) {
	s, err := t.reader.Get(sizeKey)
	if err == nil {
		b := new(big.Int).SetBytes(s).Uint64()
		return b, nil
	} else if errors.Is(err, db.ErrNotFound) {
		return 0, nil
	} else {
		return 0, err
	}
}

func (t *TreeReader) Get(key *big.Int) (*big.Int, error) {
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

func (t *TreeReader) ProveInclusion(key *big.Int) (*Proof, error) {
	i, err := t.keyIndex(key)
	if err != nil {
		return nil, err
	}
	return t.ProveIndex(i)
}

func (t *TreeReader) ProveExclusion(key *big.Int) (*Proof, error) {
	i, err := t.lowNullifierIndex(key)
	if err != nil {
		return nil, err
	}
	return t.ProveIndex(i)
}

func (t *TreeReader) ProveIndex(index uint64) (*Proof, error) {
	n, err := t.node(index)
	if err != nil {
		return nil, err
	}

	root, err := t.Root()
	if err != nil {
		return nil, err
	}
	size, err := t.Size()
	if err != nil {
		return nil, err
	}
	proof := &Proof{
		Root:     root,
		Size:     size,
		Index:    index,
		Node:     n,
		Siblings: make([]*big.Int, t.levels),
	}
	for level := t.levels; level > 0; index /= 2 {
		level--
		siblingIndex := index + 1 - (index%2)*2
		siblingHashBytes, err := t.reader.Get(t.hashKey(siblingIndex, level+1))
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		proof.Siblings[level] = new(big.Int).SetBytes(siblingHashBytes)
	}

	return proof, nil
}

func (t *TreeReader) keyIndex(key *big.Int) (uint64, error) {
	lt, err := t.reader.Get(t.indexKey(key))
	if err != nil {
		return 0, err
	}
	return new(big.Int).SetBytes(lt).Uint64(), nil
}

func (t *TreeReader) lowNullifierIndex(key *big.Int) (uint64, error) {
	_, lt, err := t.reader.GetLT(t.indexKey(key), t.zeroIndexKey())
	if err != nil {
		return 0, err
	}
	return new(big.Int).SetBytes(lt).Uint64(), nil
}

func (t *TreeReader) node(index uint64) (*Node, error) {
	b, err := t.reader.Get(t.nodeKey(new(big.Int).SetUint64(index)))
	if errors.Is(err, db.ErrNotFound) && index == 0 {
		// initial state
		return t.initialStateNode(), nil
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

func (t *TreeReader) initialStateNode() *Node {
	return &Node{
		Key:     new(big.Int),
		Value:   new(big.Int),
		NextKey: new(big.Int),
	}
}

func (t *TreeReader) indexKey(key *big.Int) []byte {
	b := key.Bytes()
	prefix := make([]byte, 1+int(t.feLen)-len(b))
	prefix[0] = indexKeyPrefix
	return append(prefix, b...)
}

func (t *TreeReader) zeroIndexKey() []byte {
	b := make([]byte, 1+int(t.feLen))
	b[0] = indexKeyPrefix
	return b
}

func (t *TreeReader) hashKey(index, level uint64) []byte {
	one := big.NewInt(1)
	position := new(big.Int).Lsh(one, uint(t.levels+1))
	position.Sub(position, new(big.Int).Lsh(one, uint(level+1)))
	position.Add(position, new(big.Int).SetUint64(index))
	return append([]byte{hashKeyPrefix}, position.Bytes()...)
}

func (t *TreeReader) nodeKey(key *big.Int) []byte {
	return append([]byte{nodeKeyPrefix}, key.Bytes()...)
}
