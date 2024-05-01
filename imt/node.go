package imt

import (
	"encoding/binary"
	"errors"
	"math/big"
)

type Node struct {
	Key     *big.Int
	Index   uint64
	Value   *big.Int
	NextKey *big.Int
}

func initialStateNode() *Node {
	return &Node{
		Key:     new(big.Int),
		Index:   0,
		Value:   new(big.Int),
		NextKey: new(big.Int),
	}
}

func bytesToNode(key *big.Int, b []byte) (*Node, error) {
	n := &Node{
		Key: key,
	}
	if len(b) < 8 {
		return nil, errors.New("invalid bytes")
	}
	n.Index = binary.BigEndian.Uint64(b)
	b = b[8:]
	if len(b) < 1 {
		return nil, errors.New("invalid bytes")
	}
	n.Value = new(big.Int).SetBytes(b[1 : 1+b[0]])
	b = b[1+b[0]:]
	if len(b) < 1 {
		return nil, errors.New("invalid bytes")
	}
	n.NextKey = new(big.Int).SetBytes(b[1 : 1+b[0]])
	return n, nil
}

func (n *Node) bytes() []byte {
	var b []byte
	b = binary.BigEndian.AppendUint64(b, n.Index)
	vb := n.Value.Bytes()
	b = append(b, byte(len(vb)))
	b = append(b, vb...)
	nkb := n.NextKey.Bytes()
	b = append(b, byte(len(nkb)))
	return append(b, nkb...)
}

func (n *Node) hash(h HashFn) (*big.Int, error) {
	return h([]*big.Int{n.Key, n.Value, n.NextKey})
}
