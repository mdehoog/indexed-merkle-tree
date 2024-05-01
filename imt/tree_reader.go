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
		initialHash, err := initialStateNode().hash(new(big.Int), t.hash)
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
	n, err := t.node(key)
	if err != nil {
		return nil, err
	}
	return n.Value, nil
}

func (t *TreeReader) ProveInclusion(key *big.Int) (*Proof, error) {
	n, err := t.node(key)
	if err != nil {
		return nil, err
	}
	return t.nodeProof(key, n)
}

func (t *TreeReader) ProveExclusion(key *big.Int) (*Proof, error) {
	k, n, err := t.lowNullifierNode(key)
	if err != nil {
		return nil, err
	}
	return t.nodeProof(k, n)
}

func (t *TreeReader) nodeProof(key *big.Int, n *Node) (*Proof, error) {
	root, err := t.Root()
	if err != nil {
		return nil, err
	}
	size, err := t.Size()
	if err != nil {
		return nil, err
	}
	siblings, err := t.proofSiblings(n)
	if err != nil {
		return nil, err
	}
	proof := &Proof{
		Root:     root,
		Size:     size,
		Key:      key,
		Node:     n,
		Siblings: siblings,
	}
	return proof, nil
}

func (t *TreeReader) proofSiblings(n *Node) ([]*big.Int, error) {
	siblings := make([]*big.Int, t.levels)
	index := n.Index
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

func (t *TreeReader) node(key *big.Int) (*Node, error) {
	b, err := t.reader.Get(t.nodeKey(key))
	if err != nil {
		return nil, err
	}
	return bytesToNode(b)
}

func (t *TreeReader) lowNullifierNode(key *big.Int) (*big.Int, *Node, error) {
	k, b, err := t.reader.GetLT(t.nodeKey(key))
	if err != nil {
		return nil, nil, err
	}
	if k == nil {
		return new(big.Int), initialStateNode(), nil
	}
	n, err := bytesToNode(b)
	return nodeKeyBytesToKey(k), n, err
}

func (t *TreeReader) nodeKey(key *big.Int) []byte {
	b := key.Bytes()
	prefix := make([]byte, 1+int(t.feLen)-len(b))
	prefix[0] = nodeKeyPrefix
	return append(prefix, b...)
}

func (t *TreeReader) hashKey(index, level uint64) []byte {
	one := big.NewInt(1)
	position := new(big.Int).Lsh(one, uint(t.levels+1))
	position.Sub(position, new(big.Int).Lsh(one, uint(level+1)))
	position.Add(position, new(big.Int).SetUint64(index))
	return append([]byte{hashKeyPrefix}, position.Bytes()...)
}

func nodeKeyBytesToKey(b []byte) *big.Int {
	return new(big.Int).SetBytes(b[1:])
}
