package imt

import (
	"math/big"
)

type Proof struct {
	Root     *big.Int
	Index    uint64
	Node     *Node
	Siblings []*big.Int
}

func (p *Proof) Valid(t *Tree) (bool, error) {
	h, err := p.Node.hash(t.hash)
	if err != nil {
		return false, err
	}
	index := p.Index
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
	return h.Cmp(p.Root) == 0, nil
}
