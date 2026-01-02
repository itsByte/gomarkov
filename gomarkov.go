package gomarkov

import (
	"encoding/binary"
	"errors"
	"math/rand"

	"github.com/cockroachdb/pebble/v2"
)

const (
	StartToken = "\u0002"
	EndToken   = "\u0003"
)

type Chain struct {
	Order   int
	Storage *PebbleStorage
}

func NewChain(order int, storage *PebbleStorage) *Chain {
	return &Chain{Order: order, Storage: storage}
}

func (c *Chain) Close() error {
	return c.Storage.db.Close()
}

// Add trains the primary order and all lower orders for Backoff support
func (c *Chain) Add(cID int64, input []string) error {
	tokens := make([]string, 0, len(input)+(c.Order*2))
	for i := 0; i < c.Order; i++ {
		tokens = append(tokens, StartToken)
	}
	tokens = append(tokens, input...)
	for i := 0; i < c.Order; i++ {
		tokens = append(tokens, EndToken)
	}

	ids := make([]uint32, len(tokens))
	for i, word := range tokens {
		id, _ := c.Storage.GetOrCreateID(word)
		ids[i] = id
	}

	batch := c.Storage.db.NewBatch()
	defer batch.Close()

	for i := 0; i < len(ids)-c.Order; i++ {
		nextID := ids[i+c.Order]
		// Train Multi-Order: e.g. Order 2 trains [A,B]->C AND [B]->C
		for o := c.Order; o >= 1; o-- {
			window := ids[i+c.Order-o : i+c.Order]
			if err := c.Storage.addTransition(batch, cID, window, nextID); err != nil {
				return err
			}
		}
	}
	err := batch.Commit(pebble.Sync)
	if err != nil {
		return err
	}
	return nil
}

// generateNextID implements the Backoff logic internally
func (c *Chain) generateNextID(cID int64, currentIDs []uint32) (uint32, error) {
	// Try the highest order, then back off to Order 1
	for len(currentIDs) > 0 {
		sumKey := c.Storage.buildKey('s', cID, currentIDs, nil)
		if val, closer, err := c.Storage.db.Get(sumKey); err == nil {
			totalWeight := binary.BigEndian.Uint32(val)
			closer.Close()

			target := rand.Intn(int(totalWeight))
			iterPrefix := c.Storage.buildKey('t', cID, currentIDs, nil)
			iter, _ := c.Storage.db.NewIter(&pebble.IterOptions{
				LowerBound: iterPrefix,
				UpperBound: upperBound(iterPrefix),
			})
			defer iter.Close()

			for iter.First(); iter.Valid(); iter.Next() {
				weight := binary.BigEndian.Uint32(iter.Value())
				target -= int(weight)
				if target < 0 {
					return binary.BigEndian.Uint32(iter.Key()[len(iter.Key())-4:]), nil
				}
			}
		}
		currentIDs = currentIDs[1:] // Backoff step
	}
	return 0, errors.New("dead end")
}

func (c *Chain) Generate(cID int64, current []string) (string, error) {
	currentIDs := make([]uint32, len(current))
	for i, word := range current {
		id, err := c.Storage.GetWordID(word)
		if err != nil {
			return "", err
		}
		currentIDs[i] = id
	}

	nextID, err := c.generateNextID(cID, currentIDs)
	if err != nil {
		return "", err
	}
	return c.Storage.GetWord(nextID)
}

func (c *Chain) GenerateAll(cID int64) ([]string, error) {
	return c.GenerateAllLimited(cID, 10000) // Default safety limit
}

func (c *Chain) GenerateAllLimited(cID int64, maxLen int) ([]string, error) {
	res := []string{}
	startID, _ := c.Storage.GetOrCreateID(StartToken)
	endID, _ := c.Storage.GetOrCreateID(EndToken)

	window := make([]uint32, c.Order)
	for i := range window {
		window[i] = startID
	}

	for range maxLen {
		nextID, err := c.generateNextID(cID, window)
		if err != nil || nextID == endID {
			break
		}

		word, _ := c.Storage.GetWord(nextID)
		res = append(res, word)
		window = append(window, nextID)[1:]
	}
	return res, nil
}

func (c *Chain) TransitionProbability(cID int64, next string, current []string) (float64, error) {
	nextID, err := c.Storage.GetWordID(next)
	if err != nil {
		return 0, err
	}
	if len(current) != c.Order {
		return 0, errors.New("N-gram length different from chain order")
	}

	currentIDs := make([]uint32, len(current))
	for i, word := range current {
		id, err := c.Storage.GetWordID(word)
		if err != nil {
			return 0, err
		}
		currentIDs[i] = id
	}

	sumKey := c.Storage.buildKey('s', cID, currentIDs, nil)
	if val, closer, err := c.Storage.db.Get(sumKey); err == nil {
		totalWeight := binary.BigEndian.Uint32(val)
		closer.Close()

		nextKey := c.Storage.buildKey('t', cID, currentIDs, &nextID)
		if val, closer, err := c.Storage.db.Get(nextKey); err == nil {
			nextWeight := binary.BigEndian.Uint32(val)
			closer.Close()

			return float64(nextWeight) / float64(totalWeight), nil

		}
	}
	return 0, nil
}
