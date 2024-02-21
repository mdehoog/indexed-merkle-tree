package imt

import (
	"github.com/consensys/gnark/frontend"
	"github.com/mdehoog/gnark-circom-smt/circuits/poseidon"
)

func VerifyExclusion(api frontend.API, enabled, root, key, lowKey, lowValue, lowNextKey, index frontend.Variable, siblings []frontend.Variable) {
	Verify(api, enabled, root, key, lowKey, lowValue, lowNextKey, index, 0, siblings)
}

func VerifyInclusion(api frontend.API, enabled, root, key, value, nextKey, index frontend.Variable, siblings []frontend.Variable) {
	Verify(api, enabled, root, key, key, value, nextKey, index, 1, siblings)
}

func Verify(api frontend.API, enabled, root, key, lowKey, value, nextKey, index, inclusion frontend.Variable, siblings []frontend.Variable) {
	prevKeyEqualsKey := api.IsZero(api.Sub(lowKey, key))
	AssertEqualIfEnabled(api, prevKeyEqualsKey, inclusion, enabled) // inclusion ? key == lowKey : key != lowKey
	AssertDifferentIfEnabled(api, key, nextKey, enabled)            // key != nextKey

	api.AssertIsLessOrEqual(api.Mul(enabled, lowKey), key)          // lowKey <= key
	nextKeyOverflow := api.Sub(nextKey, api.IsZero(nextKey))        // nextKey == 0 ? nextKey - 1 : nextKey
	api.AssertIsLessOrEqual(api.Mul(enabled, key), nextKeyOverflow) // key <= nextKey

	indexBits := api.ToBinary(index, len(siblings))
	h := poseidon.Hash(api, []frontend.Variable{lowKey, value, nextKey})
	for i := 0; i < len(siblings); i++ {
		level := len(siblings) - i - 1
		h = HashSwitcher(api, indexBits[i], h, siblings[level])
	}
	AssertEqualIfEnabled(api, h, root, enabled)
}

func AssertEqualIfEnabled(api frontend.API, a, b, enabled frontend.Variable) {
	api.AssertIsEqual(api.Mul(enabled, api.Sub(1, api.IsZero(api.Sub(a, b)))), 0)
}

func AssertDifferentIfEnabled(api frontend.API, a, b, enabled frontend.Variable) {
	api.AssertIsEqual(api.Mul(enabled, api.IsZero(api.Sub(a, b))), 0)
}

func HashSwitcher(api frontend.API, indexBit, hash, sibling frontend.Variable) frontend.Variable {
	l := api.Select(indexBit, hash, sibling)
	r := api.Select(indexBit, sibling, hash)
	return api.Select(api.IsZero(sibling), hash, poseidon.Hash(api, []frontend.Variable{l, r}))
}
