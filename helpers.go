package gomarkov

import (
	"encoding/binary"
	"io"
)

// Pair is a pair of consecutive states in a sequece
type Pair struct {
	CurrentState []string // n = order of the chain
	NextState    string   // n = 1
}

// Transition represents a transition from a set of current states to a next state
type Transition struct {
	CID        int64 // Context ID
	CurrentIDs []uint32
	NextID     uint32
	Frequency  uint32
}

type sumMerger struct{ sum uint32 }

func (m *sumMerger) MergeNewer(v []byte) error { m.sum += binary.BigEndian.Uint32(v); return nil }
func (m *sumMerger) MergeOlder(v []byte) error { m.sum += binary.BigEndian.Uint32(v); return nil }
func (m *sumMerger) Finish(base bool) ([]byte, io.Closer, error) {
	res := make([]byte, 4)
	binary.BigEndian.PutUint32(res, m.sum)
	return res, nil, nil
}

// MakePairs generates n-gram pairs of consecutive states in a sequence
func MakePairs(tokens []string, order int) []Pair {
	var pairs []Pair
	for i := 0; i < len(tokens)-order; i++ {
		pair := Pair{
			CurrentState: tokens[i : i+order],
			NextState:    tokens[i+order],
		}
		pairs = append(pairs, pair)
	}
	return pairs
}
