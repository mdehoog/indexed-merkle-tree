package imt

import (
	"errors"
	"math/big"

	"github.com/mdehoog/indexed-merkle-tree/db"
)

type HashFn func([]*big.Int) (*big.Int, error)

type TreeWriter struct {
	TreeReader
	tx db.Transaction
}

func NewTreeWriter(tx db.Transaction, levels, feLen uint64, hash HashFn) *TreeWriter {
	return &TreeWriter{
		tx: tx,
		TreeReader: TreeReader{
			reader: tx,
			levels: levels,
			feLen:  feLen,
			hash:   hash,
		},
	}
}

func (t *TreeWriter) setSize(s uint64) error {
	return t.tx.Set(sizeKey, new(big.Int).SetUint64(s).Bytes())
}

func (t *TreeWriter) Set(key, value *big.Int) (*MutateProof, error) {
	_, err := t.Get(key)
	insert := errors.Is(err, db.ErrNotFound)
	if err != nil && !insert {
		return nil, err
	}
	if insert {
		return t.Insert(key, value)
	}
	return t.Update(key, value)
}

func (t *TreeWriter) Insert(key, value *big.Int) (*MutateProof, error) {
	_, err := t.tx.Get(t.nodeKey(key))
	if err == nil {
		return nil, errors.New("key already exists")
	} else if !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}

	lowNode, err := t.lowNullifierNode(key)
	if err != nil {
		return nil, err
	}

	oldSiblings, err := t.proofSiblings(lowNode)
	if err != nil {
		return nil, err
	}

	oldRoot, err := t.Root()
	if err != nil {
		return nil, err
	}

	size, err := t.Size()
	if err != nil {
		return nil, err
	}
	size++

	err = t.setSize(size)
	if err != nil {
		return nil, err
	}

	newNode := &Node{
		Key:     key,
		Index:   size,
		Value:   value,
		NextKey: lowNode.NextKey,
	}
	_, err = t.setNode(newNode)
	if err != nil {
		return nil, err
	}

	lowNode.NextKey = key
	lowSiblings, err := t.setNode(lowNode)
	if err != nil {
		return nil, err
	}

	newRoot, err := t.Root()
	if err != nil {
		return nil, err
	}
	siblings, err := t.proofSiblings(newNode)
	if err != nil {
		return nil, err
	}

	return &MutateProof{
		OldRoot:     oldRoot,
		OldSize:     size - 1,
		OldSiblings: oldSiblings,
		NewRoot:     newRoot,
		Node:        newNode,
		Siblings:    siblings,
		LowNode:     lowNode,
		LowSiblings: lowSiblings,
		Update:      false,
	}, nil
}

func (t *TreeWriter) Update(key, value *big.Int) (*MutateProof, error) {
	oldRoot, err := t.Root()
	if err != nil {
		return nil, err
	}
	n, err := t.node(key)
	if err != nil {
		return nil, err
	}
	oldValue := n.Value
	n.Value = value
	siblings, err := t.setNode(n)
	if err != nil {
		return nil, err
	}
	newRoot, err := t.Root()
	if err != nil {
		return nil, err
	}
	size, err := t.Size()
	if err != nil {
		return nil, err
	}
	return &MutateProof{
		OldRoot:     oldRoot,
		OldSize:     size,
		OldSiblings: siblings,
		NewRoot:     newRoot,
		Node:        n,
		Siblings:    siblings,
		LowNode: &Node{
			Key:     n.Key,
			Index:   n.Index,
			Value:   oldValue,
			NextKey: n.NextKey,
		},
		LowSiblings: siblings,
		Update:      true,
	}, nil
}

func (t *TreeWriter) setNode(n *Node) ([]*big.Int, error) {
	err := t.tx.Set(t.nodeKey(n.Key), n.bytes())
	if err != nil {
		return nil, err
	}

	h, err := n.hash(t.hash)
	if err != nil {
		return nil, err
	}
	err = t.tx.Set(t.hashKey(n.Index, t.levels), h.Bytes())
	if err != nil {
		return nil, err
	}

	index := n.Index
	siblings := make([]*big.Int, t.levels)
	for level := t.levels; level > 0; {
		level--
		siblingIndex := index + 1 - (index%2)*2
		siblingHashBytes, err := t.tx.Get(t.hashKey(siblingIndex, level+1))
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

		index /= 2
		err = t.tx.Set(t.hashKey(index, level), h.Bytes())
		if err != nil {
			return nil, err
		}
		if level == 0 && index != 0 {
			return nil, errors.New("tree is over capacity")
		}
	}

	return siblings, nil
}
