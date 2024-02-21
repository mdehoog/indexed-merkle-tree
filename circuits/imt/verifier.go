package imt

import (
	"github.com/consensys/gnark/frontend"
	"github.com/mdehoog/gnark-circom-smt/circuits/poseidon"
)

func VerifyInclusion(api frontend.API, enabled, root, key, value, nextKey, index frontend.Variable, siblings []frontend.Variable) {
	Verify(api, enabled, root, key, key, value, nextKey, index, 0, siblings)
}

func VerifyExclusion(api frontend.API, enabled, root, key, prevKey, value, nextKey, index frontend.Variable, siblings []frontend.Variable) {
	Verify(api, enabled, root, key, prevKey, value, nextKey, index, 1, siblings)
}

func Verify(api frontend.API, enabled, root, key, prevKey, value, nextKey, index, exclusion frontend.Variable, siblings []frontend.Variable) {
	inclusion := api.Sub(1, exclusion)
	prevKeyEqualsKey := api.IsZero(api.Sub(prevKey, key))
	AssertEqualIfEnabled(api, prevKeyEqualsKey, inclusion, enabled) // inclusion ? key == prevKey : key != prevKey
	AssertDifferentIfEnabled(api, key, nextKey, enabled)            // key != nextKey

	api.AssertIsLessOrEqual(api.Mul(enabled, prevKey), key)         // prevKey <= key
	nextKeyOverflow := api.Sub(nextKey, api.IsZero(nextKey))        // nextKey == 0 ? nextKey - 1 : nextKey
	api.AssertIsLessOrEqual(api.Mul(enabled, key), nextKeyOverflow) // key <= nextKey

	indexBin := api.ToBinary(index, len(siblings))
	h := poseidon.Hash(api, []frontend.Variable{prevKey, value, nextKey})
	for level := len(siblings) - 1; level >= 0; level-- {
		i := indexBin[len(siblings)-1-level]
		l := api.Select(i, h, siblings[level])
		r := api.Select(i, siblings[level], h)
		h = api.Select(api.IsZero(siblings[level]), h, poseidon.Hash(api, []frontend.Variable{l, r}))
	}
	AssertEqualIfEnabled(api, h, root, enabled)
}

func AssertEqualIfEnabled(api frontend.API, a, b, enabled frontend.Variable) {
	api.AssertIsEqual(api.Mul(enabled, api.Sub(1, api.IsZero(api.Sub(a, b)))), 0)
}

func AssertDifferentIfEnabled(api frontend.API, a, b, enabled frontend.Variable) {
	api.AssertIsEqual(api.Mul(enabled, api.IsZero(api.Sub(a, b))), 0)
}
