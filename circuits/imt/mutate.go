package imt

import (
	"github.com/consensys/gnark/frontend"
	"github.com/mdehoog/gnark-circom-smt/circuits/poseidon"
)

type Mutate struct {
	Enabled     frontend.Variable
	OldRoot     frontend.Variable
	Key         frontend.Variable
	Value       frontend.Variable
	NextKey     frontend.Variable
	Index       frontend.Variable
	Siblings    []frontend.Variable
	LowKey      frontend.Variable
	LowValue    frontend.Variable
	LowIndex    frontend.Variable
	LowSiblings []frontend.Variable
	Update      frontend.Variable
}

func (p Mutate) NewRoot(api frontend.API) frontend.Variable {
	if len(p.Siblings) != len(p.LowSiblings) {
		panic("sibling length mismatch")
	}

	updateAndEnabled := api.And(p.Update, p.Enabled)
	assertEqualIfEnabled(api, p.Key, p.LowKey, updateAndEnabled)
	assertEqualIfEnabled(api, p.Index, p.LowIndex, updateAndEnabled)
	lowValueUpdate := api.Select(p.Update, p.Value, p.LowValue)

	lowNextKey := api.Select(p.Update, p.NextKey, p.Key)
	h := updateNode(api, p.Key, p.Value, p.NextKey, p.Index, p.Siblings)
	lowH := updateNode(api, p.LowKey, lowValueUpdate, lowNextKey, p.LowIndex, p.LowSiblings)

	assertEqualIfEnabled(api, h, lowH, p.Enabled)

	return api.Select(p.Enabled, h, p.OldRoot)
}

type MutateWithVerify struct {
	Mutate
	OldSiblings []frontend.Variable
}

func (p MutateWithVerify) NewRoot(api frontend.API) frontend.Variable {
	if len(p.Siblings) != len(p.OldSiblings) {
		panic("sibling length mismatch")
	}
	Verify{
		Enabled:   p.Enabled,
		Root:      p.OldRoot,
		Key:       p.Key,
		Value:     p.LowValue,
		Index:     p.LowIndex,
		NextKey:   p.NextKey,
		LowKey:    p.LowKey,
		Inclusion: p.Update,
		Siblings:  p.OldSiblings,
	}.Run(api)
	return p.Mutate.NewRoot(api)
}

func updateNode(api frontend.API, key, value, nextKey, index frontend.Variable, siblings []frontend.Variable) frontend.Variable {
	indexBits := api.ToBinary(index, len(siblings))
	h := poseidon.Hash(api, []frontend.Variable{key, value, nextKey})
	for i := 0; i < len(siblings); i++ {
		level := len(siblings) - i - 1
		h = hashSwitcher(api, indexBits[i], h, siblings[level])
	}
	return h
}
