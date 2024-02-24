package imt

import (
	"github.com/consensys/gnark/frontend"
	"github.com/mdehoog/poseidon/circuits/poseidon"
)

func assertEqualIfEnabled(api frontend.API, a, b, enabled frontend.Variable) {
	api.AssertIsEqual(api.Mul(enabled, api.Sub(1, api.IsZero(api.Sub(a, b)))), 0)
}

func assertDifferentIfEnabled(api frontend.API, a, b, enabled frontend.Variable) {
	api.AssertIsEqual(api.Mul(enabled, api.IsZero(api.Sub(a, b))), 0)
}

func hashSwitcher(api frontend.API, indexBit, hash, sibling frontend.Variable) frontend.Variable {
	l := api.Select(indexBit, hash, sibling)
	r := api.Select(indexBit, sibling, hash)
	return api.Select(api.IsZero(sibling), hash, poseidon.Hash(api, []frontend.Variable{l, r}))
}
