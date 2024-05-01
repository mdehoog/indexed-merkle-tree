package imt

import (
	"math/big"
)

type Proof struct {
	Root     *big.Int
	Size     uint64
	Node     *Node
	Siblings []*big.Int
}

func (p *Proof) Valid(t *TreeReader) (bool, error) {
	h, err := p.Node.hash(t.hash)
	if err != nil {
		return false, err
	}
	index := p.Node.Index
	for level := t.levels; level > 0; index /= 2 {
		level--
		if p.Siblings[level].Cmp(big.NewInt(0)) != 0 {
			if index%2 == 0 {
				h, err = t.hash([]*big.Int{p.Siblings[level], h})
			} else {
				h, err = t.hash([]*big.Int{h, p.Siblings[level]})
			}
			if err != nil {
				return false, err
			}
		}
	}
	h, err = t.hash([]*big.Int{h, new(big.Int).SetUint64(p.Size)})
	if err != nil {
		return false, err
	}
	return h.Cmp(p.Root) == 0, nil
}

type MutateProof struct {
	OldRoot     *big.Int
	OldSize     uint64
	OldSiblings []*big.Int
	NewRoot     *big.Int
	Node        *Node
	Siblings    []*big.Int
	LowNode     *Node // LowNode.Value == OldValue for updates
	LowSiblings []*big.Int
	Update      bool
}

func (p *MutateProof) UpdateVariable() int {
	if p.Update {
		return 1
	}
	return 0
}
