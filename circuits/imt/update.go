package imt

import "github.com/consensys/gnark/frontend"

type Update struct {
	Enabled  frontend.Variable
	OldRoot  frontend.Variable
	Size     frontend.Variable
	Key      frontend.Variable
	Value    frontend.Variable
	NextKey  frontend.Variable
	Index    frontend.Variable
	Siblings []frontend.Variable
}

func (p Update) NewRoot(api frontend.API) frontend.Variable {
	h := updateNode(api, p.Size, p.Key, p.Value, p.NextKey, p.Index, p.Siblings)
	return api.Select(p.Enabled, h, p.OldRoot)
}

type UpdateWithVerify struct {
	Update
	OldValue frontend.Variable
}

func (p UpdateWithVerify) NewRoot(api frontend.API) frontend.Variable {
	Verify{
		Enabled:   p.Enabled,
		Root:      p.OldRoot,
		Size:      p.Size,
		Key:       p.Key,
		Value:     p.OldValue,
		Index:     p.Index,
		NextKey:   p.NextKey,
		LowKey:    p.Key,
		Inclusion: 1,
		Siblings:  p.Siblings,
	}.Run(api)
	return p.Update.NewRoot(api)
}
