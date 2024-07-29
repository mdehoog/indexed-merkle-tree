package imt

import (
	"fmt"
	"math/big"
)

type Proof interface {
	Root() *big.Int
	Size() uint64
	Node() Node
	Siblings() []*big.Int
	Valid(t TreeReader) (bool, error)
}

type proof struct {
	root     *big.Int
	size     uint64
	node     Node
	siblings []*big.Int
}

var _ Proof = (*proof)(nil)

func (p *proof) Root() *big.Int {
	return p.root
}

func (p *proof) Size() uint64 {
	return p.size
}

func (p *proof) Node() Node {
	return p.node
}

func (p *proof) Siblings() []*big.Int {
	return p.siblings
}

func (p *proof) Valid(t TreeReader) (bool, error) {
	h, err := p.node.Hash(t.Hash)
	if err != nil {
		return false, err
	}
	index := p.node.Index()
	for level := t.Levels(); level > 0; index /= 2 {
		level--
		if p.siblings[level].Cmp(big.NewInt(0)) != 0 {
			if index%2 == 0 {
				h, err = t.Hash([]*big.Int{h, p.siblings[level]})
			} else {
				h, err = t.Hash([]*big.Int{p.siblings[level], h})
			}
			if err != nil {
				return false, err
			}
		}
	}
	h, err = t.Hash([]*big.Int{h, new(big.Int).SetUint64(p.size)})
	if err != nil {
		return false, err
	}
	return h.Cmp(p.root) == 0, nil
}

func (p *proof) String() string {
	return fmt.Sprintf("Proof{Root: %s, Size: %d, Node: %s, Siblings: %v}", p.root, p.size, p.node, p.siblings)
}

type MutateProof interface {
	OldRoot() *big.Int
	OldSize() uint64
	OldSiblings() []*big.Int
	NewRoot() *big.Int
	Node() Node
	Siblings() []*big.Int
	LowNode() Node
	LowSiblings() []*big.Int
	Update() bool
	UpdateVariable() int
}

type mutateProof struct {
	oldRoot     *big.Int
	oldSize     uint64
	oldSiblings []*big.Int
	newRoot     *big.Int
	node        Node
	siblings    []*big.Int
	lowNode     Node // LowNode.Value == OldValue for updates
	lowSiblings []*big.Int
	update      bool
}

var _ MutateProof = (*mutateProof)(nil)

func (p *mutateProof) OldRoot() *big.Int {
	return p.oldRoot
}

func (p *mutateProof) OldSize() uint64 {
	return p.oldSize
}

func (p *mutateProof) OldSiblings() []*big.Int {
	return p.oldSiblings
}

func (p *mutateProof) NewRoot() *big.Int {
	return p.newRoot
}

func (p *mutateProof) Node() Node {
	return p.node
}

func (p *mutateProof) Siblings() []*big.Int {
	return p.siblings
}

func (p *mutateProof) LowNode() Node {
	return p.lowNode
}

func (p *mutateProof) LowSiblings() []*big.Int {
	return p.lowSiblings
}

func (p *mutateProof) Update() bool {
	return p.update
}

func (p *mutateProof) UpdateVariable() int {
	if p.update {
		return 1
	}
	return 0
}

func (p *mutateProof) String() string {
	return fmt.Sprintf("MutateProof{OldRoot: %s, OldSize: %d, OldSiblings: %v, NewRoot: %s, Node: %s, Siblings: %v, LowNode: %s, LowSiblings: %v, Update: %t}", p.oldRoot, p.oldSize, p.oldSiblings, p.newRoot, p.node, p.siblings, p.lowNode, p.lowSiblings, p.update)
}
