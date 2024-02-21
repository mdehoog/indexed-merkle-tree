package imt

import "github.com/consensys/gnark/frontend"

type Insert struct {
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
}

func (p Insert) NewRoot(api frontend.API) frontend.Variable {
	return Mutate{
		Enabled:     p.Enabled,
		OldRoot:     p.OldRoot,
		Key:         p.Key,
		Value:       p.Value,
		NextKey:     p.NextKey,
		Index:       p.Index,
		Siblings:    p.Siblings,
		LowKey:      p.LowKey,
		LowValue:    p.LowValue,
		LowIndex:    p.LowIndex,
		LowSiblings: p.LowSiblings,
		Update:      0,
	}.NewRoot(api)
}

type InsertWithVerify struct {
	Insert
	OldSiblings []frontend.Variable
}

func (p InsertWithVerify) NewRoot(api frontend.API) frontend.Variable {
	return MutateWithVerify{
		Mutate: Mutate{
			Enabled:     p.Enabled,
			OldRoot:     p.OldRoot,
			Key:         p.Key,
			Value:       p.Value,
			NextKey:     p.NextKey,
			Index:       p.Index,
			Siblings:    p.Siblings,
			LowKey:      p.LowKey,
			LowValue:    p.LowValue,
			LowIndex:    p.LowIndex,
			LowSiblings: p.LowSiblings,
			Update:      0,
		},
		OldSiblings: p.OldSiblings,
	}.NewRoot(api)
}
