# indexed-merkle-tree

Golang + Gnark implementation of an indexed merkle tree, as described in
[Aztec's documentation](https://docs.aztec.network/learn/concepts/storage/trees/indexed_merkle_tree).

There are two differences between the one described in the Aztec documentation and this implementation:
1. The Aztec version stores the `[value, nextIndex, nextValue]` in the leaf nodes, whereas this implementation 
   stores `[key, value, nextKey]`.
2. This implementation performs an additional hash of the root node with the `size` of the tree to generate the
   final root hash. This is done to simplify insertion of new nodes into the tree, which can only be inserted
   at index `size` (i.e. `max(index)+1`).

## Usage

### Golang

```golang
levels := 64

temp, _ := os.MkdirTemp("", "*")
pDb, _ := pebble.Open(temp, &pebble.Options{})
imtDb := db.NewPebble(pDb)
tx := imtDb.NewTransaction()
tree := imt.NewTreeWriter(tx, levels, fr.Bytes, poseidon.Hash[*fr.Element])

key := big.NewInt(123)
exclusionProof, _ := tree.ProveExclusion(key)
insertProof, _ := tree.Insert(key, big.NewInt(456))
updateProof, _ := tree.Update(key, big.NewInt(789))
inclusionProof, _ := tree.ProveInclusion(key)
```

### Gnark verification

Exclusion proof:
```golang
type ExclusionCircuit struct {
	Root, Size, Key, Index, LowKey, LowValue, LowNextKey frontend.Variable
	Siblings                                             []frontend.Variable
}

func (c *ExclusionCircuit) Define(api frontend.API) error {
	imt.Exclusion{
		Enabled:    1,
		Root:       c.Root,
		Size:       c.Size,
		Key:        c.Key,
		Index:      c.Index,
		LowKey:     c.LowKey,
		LowValue:   c.LowValue,
		LowNextKey: c.LowNextKey,
		Siblings:   c.Siblings,
	}.Run(api)
	return nil
}
```

Inclusion proof:
```golang
type InclusionCircuit struct {
	Root, Size, Key, Value, Index, NextKey frontend.Variable
	Siblings                               []frontend.Variable
}

func (c *InclusionCircuit) Define(api frontend.API) error {
	imt.Inclusion{
		Enabled:  1,
		Root:     c.Root,
		Size:     c.Size,
		Key:      c.Key,
		Value:    c.Value,
		Index:    c.Index,
		NextKey:  c.NextKey,
		Siblings: c.Siblings,
	}.Run(api)
	return nil
}
```

Both exclusion or inclusion:
```golang
type VerifyCircuit struct {
	Root, Size, Key, LowKey, Value, NextKey, Index, Inclusion frontend.Variable
	Siblings                                                  []frontend.Variable
}

func (c *VerifyCircuit) Define(api frontend.API) error {
	imt.Verify{
		Enabled:   1,
		Root:      c.Root,
		Size:      c.Size,
		Key:       c.Key,
		Value:     c.Value, // LowValue for exclusion
		Index:     c.Index,
		NextKey:   c.NextKey, // LowNextKey for exclusion
		LowKey:    c.LowKey,
		Siblings:  c.Siblings,
		Inclusion: c.Inclusion,
	}.Run(api)
	return nil
}
```

### Gnark tree modification:

Insert:
```golang
type InsertCircuit struct {
	OldRoot, OldSize, NewRoot, Key, Value, NextKey, LowKey, LowValue, LowIndex frontend.Variable
	OldSiblings, Siblings, LowSiblings                                         []frontend.Variable
}

func (c *InsertCircuit) Define(api frontend.API) error {
	newRoot := imt.InsertWithVerify{
		Insert: imt.Insert{
			Enabled:     1,
			OldSize:     c.OldSize,
			OldRoot:     c.OldRoot,
			Key:         c.Key,
			Value:       c.Value,
			NextKey:     c.NextKey,
			Siblings:    c.Siblings,
			LowKey:      c.LowKey,
			LowValue:    c.LowValue,
			LowIndex:    c.LowIndex,
			LowSiblings: c.LowSiblings,
		},
		OldSiblings: c.OldSiblings,
	}.NewRoot(api)
	api.AssertIsEqual(newRoot, c.NewRoot)
	return nil
}
```

Update:
```golang
type UpdateCircuit struct {
	Size, OldRoot, NewRoot, Key, Value, NextKey, Index, OldValue frontend.Variable
	Siblings                                                     []frontend.Variable
}

func (c *UpdateCircuit) Define(api frontend.API) error {
	newRoot := imt.UpdateWithVerify{
		Update: imt.Update{
			Enabled:  1,
			Size:     c.Size,
			OldRoot:  c.OldRoot,
			Key:      c.Key,
			Value:    c.Value,
			NextKey:  c.NextKey,
			Index:    c.Index,
			Siblings: c.Siblings,
		},
		OldValue: c.OldValue,
	}.NewRoot(api)
	api.AssertIsEqual(newRoot, c.NewRoot)
	return nil
}
```

Insert or update:
```golang
type MutateCircuit struct {
   OldSize, OldRoot, NewRoot, Key, Value, NextKey, LowKey, LowValue, LowIndex, Update frontend.Variable
   OldSiblings, Siblings, LowSiblings                                                 []frontend.Variable
}

func (c *MutateCircuit) Define(api frontend.API) error {
   newRoot := imt.MutateWithVerify{
	  Mutate: imt.Mutate{
		 Enabled:     1,
		 OldSize:     c.OldSize,
		 OldRoot:     c.OldRoot,
		 Key:         c.Key,
		 Value:       c.Value,
		 NextKey:     c.NextKey,
		 Siblings:    c.Siblings,
		 LowKey:      c.LowKey,
		 LowValue:    c.LowValue,
		 LowIndex:    c.LowIndex,
		 LowSiblings: c.LowSiblings,
		 Update:      c.Update,
	  },
	  OldSiblings: c.OldSiblings,
   }.NewRoot(api)
   api.AssertIsEqual(newRoot, c.NewRoot)
   return nil
}
```

## Database

The `db` package provides an interface for the indexed merkle tree to interact with a database. The `pebble` package
provides an implementation of this interface using the `pebble` database.

If you plan to share the database with other data, please be mindful of avoiding collisions with the data that the
indexed merkle tree stores. In particular, do not store any keys that are prefixed with a `1` byte. This namespace
is reserved for the indexed merkle tree to ensure the low nullifier can be looked up correctly.
