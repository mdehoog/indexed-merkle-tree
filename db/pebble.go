package db

import (
	"errors"
	"io"

	"github.com/cockroachdb/pebble"
)

type Pebble struct {
	db           *pebble.DB
	writeOptions *pebble.WriteOptions
}

type pebbleGetter interface {
	Get([]byte) ([]byte, io.Closer, error)
	NewIter(*pebble.IterOptions) (*pebble.Iterator, error)
}

var _ Database = (*Pebble)(nil)

func NewPebble(db *pebble.DB) *Pebble {
	return &Pebble{
		db:           db,
		writeOptions: &pebble.WriteOptions{Sync: true},
	}
}

func (p *Pebble) NewTransaction() Transaction {
	return &pebbleTransaction{
		batch:        p.db.NewIndexedBatch(),
		writeOptions: p.writeOptions,
	}
}

func (p *Pebble) Get(key []byte) ([]byte, error) {
	return get(key, p.db)
}

func (p *Pebble) GetLT(key []byte) ([]byte, []byte, error) {
	return getLT(key, p.db)
}

func (p *Pebble) Close() error {
	return p.db.Close()
}

type pebbleTransaction struct {
	batch        *pebble.Batch
	writeOptions *pebble.WriteOptions
}

var _ Transaction = (*pebbleTransaction)(nil)

func (p *pebbleTransaction) Get(key []byte) ([]byte, error) {
	return get(key, p.batch)
}

func (p *pebbleTransaction) GetLT(key []byte) ([]byte, []byte, error) {
	return getLT(key, p.batch)
}

func (p *pebbleTransaction) Set(key []byte, value []byte) error {
	return p.batch.Set(key, value, p.writeOptions)
}

func (p *pebbleTransaction) Commit() error {
	if p.batch == nil {
		return errors.New("commit: transaction already committed")
	}
	err := p.batch.Commit(p.writeOptions)
	p.batch = nil
	return err
}

func (p *pebbleTransaction) Discard() {
	if p.batch == nil {
		return
	}
	_ = p.batch.Close()
	p.batch = nil
}

func (p *pebbleTransaction) Apply(transaction Transaction) error {
	otherPebble, ok := transaction.(*pebbleTransaction)
	if !ok {
		return errors.New("apply: incompatible transaction types")
	}
	return p.batch.Apply(otherPebble.batch, nil)
}

func get(key []byte, g pebbleGetter) ([]byte, error) {
	dat, closer, err := g.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	ret := make([]byte, len(dat))
	copy(ret, dat)
	_ = closer.Close()
	return ret, nil
}

func getLT(key []byte, g pebbleGetter) ([]byte, []byte, error) {
	iter, err := g.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = iter.Close()
	}()
	if !iter.SeekLT(key) {
		return nil, nil, nil
	}
	v, err := iter.ValueAndErr()
	if err != nil {
		return nil, nil, err
	}
	return iter.Key(), v, nil
}
