package imt

import (
	"github.com/consensys/gnark/frontend"
	"github.com/mdehoog/poseidon/circuits/poseidon"
)

type Verify struct {
	Enabled   frontend.Variable
	Root      frontend.Variable
	Size      frontend.Variable
	Key       frontend.Variable
	Value     frontend.Variable // exclusion: use LowValue
	Index     frontend.Variable
	NextKey   frontend.Variable // exclusion: use LowNextKey
	LowKey    frontend.Variable
	Siblings  []frontend.Variable
	Inclusion frontend.Variable
}

func (v Verify) Run(api frontend.API) {
	prevKeyEqualsKey := api.IsZero(api.Sub(v.LowKey, v.Key))
	assertEqualIfEnabled(api, prevKeyEqualsKey, v.Inclusion, v.Enabled) // inclusion ? key == lowKey : key != lowKey
	assertDifferentIfEnabled(api, v.Key, v.NextKey, v.Enabled)          // key != nextKey

	api.AssertIsLessOrEqual(api.Mul(v.Enabled, v.LowKey), v.Key)        // lowKey <= key
	nextKeyOverflow := api.Sub(v.NextKey, api.IsZero(v.NextKey))        // nextKey == 0 ? nextKey - 1 : nextKey
	api.AssertIsLessOrEqual(api.Mul(v.Enabled, v.Key), nextKeyOverflow) // key <= nextKey

	indexBits := api.ToBinary(v.Index, len(v.Siblings))
	h := poseidon.Hash(api, []frontend.Variable{v.LowKey, v.Value, v.NextKey})
	for i := 0; i < len(v.Siblings); i++ {
		level := len(v.Siblings) - i - 1
		h = hashSwitcher(api, indexBits[i], h, v.Siblings[level])
	}
	h = poseidon.Hash(api, []frontend.Variable{h, v.Size})
	assertEqualIfEnabled(api, h, v.Root, v.Enabled)
}
