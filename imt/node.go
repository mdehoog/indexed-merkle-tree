package imt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)

type HashFn func([]*big.Int) (*big.Int, error)

type Node interface {
	Key() *big.Int
	Index() uint64
	Value() *big.Int
	NextKey() *big.Int
	Hash(HashFn) (*big.Int, error)
	Bytes() []byte
}

type node struct {
	key     *big.Int
	index   uint64
	value   *big.Int
	nextKey *big.Int
}

func initialStateNode() Node {
	return &node{
		key:     new(big.Int),
		index:   0,
		value:   new(big.Int),
		nextKey: new(big.Int),
	}
}

func bytesToNode(key *big.Int, b []byte) (Node, error) {
	n := &node{
		key: key,
	}
	if len(b) < 8 {
		return nil, errors.New("invalid bytes")
	}
	n.index = binary.BigEndian.Uint64(b)
	b = b[8:]
	if len(b) < 1 {
		return nil, errors.New("invalid bytes")
	}
	n.value = new(big.Int).SetBytes(b[1 : 1+b[0]])
	b = b[1+b[0]:]
	if len(b) < 1 {
		return nil, errors.New("invalid bytes")
	}
	n.nextKey = new(big.Int).SetBytes(b[1 : 1+b[0]])
	return n, nil
}

func (n *node) Key() *big.Int {
	return n.key
}

func (n *node) Index() uint64 {
	return n.index
}

func (n *node) Value() *big.Int {
	return n.value
}

func (n *node) NextKey() *big.Int {
	return n.nextKey
}

func (n *node) Hash(fn HashFn) (*big.Int, error) {
	return fn([]*big.Int{n.key, n.value, n.nextKey})
}

func (n *node) Bytes() []byte {
	var b []byte
	b = binary.BigEndian.AppendUint64(b, n.index)
	vb := n.value.Bytes()
	b = append(b, byte(len(vb)))
	b = append(b, vb...)
	nkb := n.nextKey.Bytes()
	b = append(b, byte(len(nkb)))
	return append(b, nkb...)
}

func (n *node) String() string {
	return fmt.Sprintf("Node{key: %s, index: %d, value: %s, nextKey: %s}", n.key, n.index, n.value, n.nextKey)
}
