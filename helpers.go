package gomarkov

import (
	"sort"
	"strings"
)

// Pair is a pair of consecutive states in a sequece
type Pair struct {
	CurrentState NGram  // n = order of the chain
	NextState    string // n = 1
}

// NGram is a array of words
type NGram []string

type sparseArray map[int]int

func (ngram NGram) key() string {
	return strings.Join(ngram, "_")
}

func (s sparseArray) orderedKeys() []int {
	keys := make([]int, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func (s sparseArray) sum() int {
	sum := 0
	for _, count := range s {
		sum += count
	}
	return sum
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func array(value string, count int) []string {
	arr := make([]string, count)
	for i := range arr {
		arr[i] = value
	}
	return arr
}

// MakePairs generates n-gram pairs of consecutive states in a sequence
func MakePairs(tokens []string, order int) []Pair {
	var pairs []Pair
	pair := Pair{
		CurrentState: tokens[0:1],
		NextState:    tokens[1],
	}
	pairs = append(pairs, pair)
	for i := 1; i < len(tokens)-order-1; i++ {
		pair := Pair{
			CurrentState: tokens[i : i+order],
			NextState:    tokens[i+order],
		}
		pairs = append(pairs, pair)
	}
	pair = Pair{
		CurrentState: tokens[len(tokens)-2 : len(tokens)-1],
		NextState:    tokens[len(tokens)-1],
	}
	pairs = append(pairs, pair)

	return pairs
}
