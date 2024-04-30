package db

import "errors"

var ErrNotFound = errors.New("not found")

type Database interface {
	Reader
	NewTransaction() Transaction
	Close() error
}

type Reader interface {
	// Get retrieves the value for the given key. If the key does not
	// exist, returns the error ErrNotFound
	Get(key []byte) ([]byte, error)

	// GetLT retrieves the key/value less than the given key.
	GetLT(key []byte) ([]byte, []byte, error)
}

type Transaction interface {
	Reader
	Set(key []byte, value []byte) error
	Commit() error
	Discard()
	Apply(Transaction) error
}
