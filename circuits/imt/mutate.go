package imt

import (
	"github.com/consensys/gnark/frontend"
	"github.com/mdehoog/poseidon/circuits/poseidon"
)

type Mutate struct {
	Enabled     frontend.Variable
	OldSize     frontend.Variable // updates: use Size
	OldRoot     frontend.Variable
	Key         frontend.Variable
	Value       frontend.Variable
	NextKey     frontend.Variable
	Siblings    []frontend.Variable
	LowKey      frontend.Variable   // updates: same as Key
	LowValue    frontend.Variable   // updates: use OldValue
	LowIndex    frontend.Variable   // updates: use Index
	LowSiblings []frontend.Variable // updates: same as Siblings
	Update      frontend.Variable
}

func (p Mutate) NewRoot(api frontend.API) frontend.Variable {
	if len(p.Siblings) != len(p.LowSiblings) {
		panic("sibling length mismatch")
	}

	updateAndEnabled := api.And(p.Update, p.Enabled)
	assertEqualIfEnabled(api, p.Key, p.LowKey, updateAndEnabled)
	lowValueUpdate := api.Select(p.Update, p.Value, p.LowValue)
	size := api.Add(p.OldSize, api.IsZero(p.Update))
	index := api.Select(p.Update, p.LowIndex, size)

	lowNextKey := api.Select(p.Update, p.NextKey, p.Key)
	h := updateNode(api, size, p.Key, p.Value, p.NextKey, index, p.Siblings)
	lowH := updateNode(api, size, p.LowKey, lowValueUpdate, lowNextKey, p.LowIndex, p.LowSiblings)

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
		Size:      p.OldSize,
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

func updateNode(api frontend.API, size, key, value, nextKey, index frontend.Variable, siblings []frontend.Variable) frontend.Variable {
	indexBits := api.ToBinary(index, len(siblings))
	h := poseidon.Hash(api, []frontend.Variable{key, value, nextKey})
	for i := 0; i < len(siblings); i++ {
		level := len(siblings) - i - 1
		h = hashSwitcher(api, indexBits[i], h, siblings[level])
	}
	h = poseidon.Hash(api, []frontend.Variable{h, size})
	return h
}
