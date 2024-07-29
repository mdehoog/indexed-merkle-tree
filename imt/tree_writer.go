package imt

import (
	"errors"
	"math/big"

	"github.com/mdehoog/indexed-merkle-tree/db"
)

type TreeWriter interface {
	TreeReader
	Set(key, value *big.Int) (MutateProof, error)
	Insert(key, value *big.Int) (MutateProof, error)
	Update(key, value *big.Int) (MutateProof, error)
}

type treeWriter struct {
	*treeReader
	tx db.Transaction
}

func NewTreeWriter(tx db.Transaction, levels, feLen uint64, hash HashFn) TreeWriter {
	return &treeWriter{
		tx: tx,
		treeReader: &treeReader{
			reader: tx,
			levels: levels,
			feLen:  feLen,
			hash:   hash,
		},
	}
}

func (t *treeWriter) setSize(s uint64) error {
	return t.tx.Set(sizeKey, new(big.Int).SetUint64(s).Bytes())
}

func (t *treeWriter) Set(key, value *big.Int) (MutateProof, error) {
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

func (t *treeWriter) Insert(key, value *big.Int) (MutateProof, error) {
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

	oldSiblings, err := t.proveSiblings(lowNode)
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

	newNode := &node{
		key:     key,
		index:   size,
		value:   value,
		nextKey: lowNode.NextKey(),
	}
	_, err = t.setNode(newNode)
	if err != nil {
		return nil, err
	}

	lowNode = &node{
		key:     lowNode.Key(),
		index:   lowNode.Index(),
		value:   lowNode.Value(),
		nextKey: key,
	}
	lowSiblings, err := t.setNode(lowNode)
	if err != nil {
		return nil, err
	}

	newRoot, err := t.Root()
	if err != nil {
		return nil, err
	}
	siblings, err := t.proveSiblings(newNode)
	if err != nil {
		return nil, err
	}

	return &mutateProof{
		oldRoot:     oldRoot,
		oldSize:     size - 1,
		oldSiblings: oldSiblings,
		newRoot:     newRoot,
		node:        newNode,
		siblings:    siblings,
		lowNode:     lowNode,
		lowSiblings: lowSiblings,
		update:      false,
	}, nil
}

func (t *treeWriter) Update(key, value *big.Int) (MutateProof, error) {
	oldRoot, err := t.Root()
	if err != nil {
		return nil, err
	}
	n, err := t.node(key)
	if err != nil {
		return nil, err
	}
	oldValue := n.Value()
	n = &node{
		key:     key,
		index:   n.Index(),
		value:   value,
		nextKey: n.NextKey(),
	}
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
	return &mutateProof{
		oldRoot:     oldRoot,
		oldSize:     size,
		oldSiblings: siblings,
		newRoot:     newRoot,
		node:        n,
		siblings:    siblings,
		lowNode: &node{
			key:     n.Key(),
			index:   n.Index(),
			value:   oldValue,
			nextKey: n.NextKey(),
		},
		lowSiblings: siblings,
		update:      true,
	}, nil
}

func (t *treeWriter) setNode(n *node) ([]*big.Int, error) {
	err := t.tx.Set(t.nodeKey(n.Key()), n.bytes())
	if err != nil {
		return nil, err
	}

	h, err := n.Hash(t.hash)
	if err != nil {
		return nil, err
	}
	err = t.tx.Set(t.hashKey(n.Index(), t.levels), h.Bytes())
	if err != nil {
		return nil, err
	}

	index := n.Index()
	siblings := make([]*big.Int, t.levels)
	for level := t.levels; level > 0; {
		level--
		siblingIndex := index + 1 - (index%2)*2
		siblingHashBytes, err := t.tx.Get(t.hashKey(siblingIndex, level+1))
		siblings[level] = new(big.Int).SetBytes(siblingHashBytes)
		if err == nil {
			if index%2 == 0 {
				h, err = t.hash([]*big.Int{h, siblings[level]})
			} else {
				h, err = t.hash([]*big.Int{siblings[level], h})
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
