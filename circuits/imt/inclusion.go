package imt

import "github.com/consensys/gnark/frontend"

type Inclusion struct {
	Enabled  frontend.Variable
	Root     frontend.Variable
	Size     frontend.Variable
	Key      frontend.Variable
	Value    frontend.Variable
	Index    frontend.Variable
	NextKey  frontend.Variable
	Siblings []frontend.Variable
}

func (v Inclusion) Run(api frontend.API) {
	Verify{
		Enabled:   v.Enabled,
		Root:      v.Root,
		Size:      v.Size,
		Key:       v.Key,
		Value:     v.Value,
		Index:     v.Index,
		NextKey:   v.NextKey,
		LowKey:    v.Key,
		Siblings:  v.Siblings,
		Inclusion: 1,
	}.Run(api)
}
