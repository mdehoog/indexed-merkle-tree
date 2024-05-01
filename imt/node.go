package imt

import (
	"encoding/binary"
	"errors"
	"math/big"
)

type Node struct {
	Index   uint64
	Value   *big.Int
	NextKey *big.Int
}

func initialStateNode() *Node {
	return &Node{
		Index:   0,
		Value:   new(big.Int),
		NextKey: new(big.Int),
	}
}

func bytesToNode(b []byte) (*Node, error) {
	n := &Node{}
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

func (n *Node) hash(key *big.Int, h HashFn) (*big.Int, error) {
	return h([]*big.Int{key, n.Value, n.NextKey})
}
