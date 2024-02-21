package imt

import "github.com/consensys/gnark/frontend"

type Exclusion struct {
	Enabled    frontend.Variable
	Root       frontend.Variable
	Key        frontend.Variable
	Index      frontend.Variable
	LowKey     frontend.Variable
	LowValue   frontend.Variable
	LowNextKey frontend.Variable
	Siblings   []frontend.Variable
}

func (v Exclusion) Run(api frontend.API) {
	Verify{
		Enabled:   v.Enabled,
		Root:      v.Root,
		Key:       v.Key,
		Value:     v.LowValue,
		Index:     v.Index,
		NextKey:   v.LowNextKey,
		LowKey:    v.LowKey,
		Siblings:  v.Siblings,
		Inclusion: 0,
	}.Run(api)
}
