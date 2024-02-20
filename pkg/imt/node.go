package imt

import (
	"encoding/binary"
	"errors"
	"math/big"
)

type Node struct {
	Key       *big.Int
	Value     *big.Int
	NextKey   *big.Int
	NextIndex uint64
}

func (n *Node) bytes() []byte {
	var b []byte
	kb := n.Key.Bytes()
	b = append(b, byte(len(kb)))
	b = append(b, kb...)
	vb := n.Value.Bytes()
	b = append(b, byte(len(vb)))
	b = append(b, vb...)
	nkb := n.NextKey.Bytes()
	b = append(b, byte(len(nkb)))
	b = append(b, nkb...)
	return binary.BigEndian.AppendUint64(b, n.NextIndex)
}

func (n *Node) fromBytes(b []byte) error {
	if len(b) < 1 {
		return errors.New("invalid bytes")
	}
	n.Key = new(big.Int).SetBytes(b[1 : 1+b[0]])
	b = b[1+b[0]:]
	if len(b) < 1 {
		return errors.New("invalid bytes")
	}
	n.Value = new(big.Int).SetBytes(b[1 : 1+b[0]])
	b = b[1+b[0]:]
	if len(b) < 1 {
		return errors.New("invalid bytes")
	}
	n.NextKey = new(big.Int).SetBytes(b[1 : 1+b[0]])
	b = b[1+b[0]:]
	n.NextIndex = binary.BigEndian.Uint64(b)
	return nil
}

func (n *Node) hash(h HashFn) (*big.Int, error) {
	return h([]*big.Int{n.Key, n.Value, n.NextKey, new(big.Int).SetUint64(n.NextIndex)})
}
