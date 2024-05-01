package imt

import (
	"errors"
	"math/big"

	"github.com/mdehoog/indexed-merkle-tree/db"
)

const nodeKeyPrefix = byte(0)
const hashKeyPrefix = byte(1)
const sizeKeyPrefix = byte(2)

var sizeKey = []byte{sizeKeyPrefix}

type TreeReader interface {
	Hash([]*big.Int) (*big.Int, error)
	Levels() uint64
	Root() (*big.Int, error)
	Size() (uint64, error)
	Get(key *big.Int) (*big.Int, error)
	ProveInclusion(key *big.Int) (Proof, error)
	ProveExclusion(key *big.Int) (Proof, error)
}

type treeReader struct {
	reader db.Reader
	levels uint64
	feLen  uint64
	hash   HashFn
}

func NewTreeReader(reader db.Reader, levels, feLen uint64, hash HashFn) TreeReader {
	return &treeReader{
		reader: reader,
		levels: levels,
		feLen:  feLen,
		hash:   hash,
	}
}

func (t *treeReader) Hash(i []*big.Int) (*big.Int, error) {
	return t.hash(i)
}

func (t *treeReader) Levels() uint64 {
	return t.levels
}

func (t *treeReader) Root() (*big.Int, error) {
	rootNodeBytes, err := t.reader.Get(t.hashKey(0, 0))
	if errors.Is(err, db.ErrNotFound) {
		// initial state: hash of empty node
		initialHash, err := initialStateNode().Hash(t.hash)
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

func (t *treeReader) Size() (uint64, error) {
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

func (t *treeReader) Get(key *big.Int) (*big.Int, error) {
	n, err := t.node(key)
	if err != nil {
		return nil, err
	}
	return n.Value(), nil
}

func (t *treeReader) ProveInclusion(key *big.Int) (Proof, error) {
	n, err := t.node(key)
	if err != nil {
		return nil, err
	}
	return t.proveNode(n)
}

func (t *treeReader) ProveExclusion(key *big.Int) (Proof, error) {
	n, err := t.lowNullifierNode(key)
	if err != nil {
		return nil, err
	}
	return t.proveNode(n)
}

func (t *treeReader) proveNode(n Node) (Proof, error) {
	root, err := t.Root()
	if err != nil {
		return nil, err
	}
	size, err := t.Size()
	if err != nil {
		return nil, err
	}
	siblings, err := t.proveSiblings(n)
	if err != nil {
		return nil, err
	}
	return &proof{
		root:     root,
		size:     size,
		node:     n,
		siblings: siblings,
	}, nil
}

func (t *treeReader) proveSiblings(n Node) ([]*big.Int, error) {
	siblings := make([]*big.Int, t.levels)
	index := n.Index()
	for level := t.levels; level > 0; index /= 2 {
		level--
		siblingIndex := index + 1 - (index%2)*2
		siblingHashBytes, err := t.reader.Get(t.hashKey(siblingIndex, level+1))
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		siblings[level] = new(big.Int).SetBytes(siblingHashBytes)
	}
	return siblings, nil
}

func (t *treeReader) node(key *big.Int) (Node, error) {
	b, err := t.reader.Get(t.nodeKey(key))
	if err != nil {
		return nil, err
	}
	return bytesToNode(key, b)
}

func (t *treeReader) lowNullifierNode(key *big.Int) (Node, error) {
	k, b, err := t.reader.GetLT(t.nodeKey(key))
	if err != nil {
		return nil, err
	}
	if k == nil {
		return initialStateNode(), nil
	}
	return bytesToNode(nodeKeyBytesToKey(k), b)
}

func (t *treeReader) nodeKey(key *big.Int) []byte {
	b := key.Bytes()
	prefix := make([]byte, 1+int(t.feLen)-len(b))
	prefix[0] = nodeKeyPrefix
	return append(prefix, b...)
}

func (t *treeReader) hashKey(index, level uint64) []byte {
	one := big.NewInt(1)
	position := new(big.Int).Lsh(one, uint(t.levels+1))
	position.Sub(position, new(big.Int).Lsh(one, uint(level+1)))
	position.Add(position, new(big.Int).SetUint64(index))
	return append([]byte{hashKeyPrefix}, position.Bytes()...)
}

func nodeKeyBytesToKey(b []byte) *big.Int {
	return new(big.Int).SetBytes(b[1:])
}
