package gomarkov

import (
	"encoding/binary"
	"sync"

	"github.com/cockroachdb/pebble/v2"
)

const (
	prefixWordToID   byte = 'w'
	prefixIDToWord   byte = 'i'
	prefixTransition byte = 't'
	prefixSum        byte = 's'
)

type PebbleStorage struct {
	db *pebble.DB
	mu sync.Mutex
}

func uint32Adder(_, value []byte) (pebble.ValueMerger, error) {
	return &sumMerger{sum: binary.BigEndian.Uint32(value)}, nil
}

// Creates a new PebbleStorage instance.
func NewPebbleStorage(path string) (*PebbleStorage, error) {
	opts := &pebble.Options{
		Merger: &pebble.Merger{Name: "uint32-adder", Merge: uint32Adder},
	}
	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, err
	}
	return &PebbleStorage{db: db}, nil
}

func (s *PebbleStorage) Close() error {
	return s.db.Close()
}

// buildKey: [Prefix(1)][cID(8)][Order(1)][IDs(N*4)][NextID(4)]
func (s *PebbleStorage) buildKey(p byte, cID int64, currentIDs []uint32, nextID *uint32) []byte {
	order := byte(len(currentIDs))
	size := 1 + 8 + 1 + (len(currentIDs) * 4)
	if nextID != nil {
		size += 4
	}
	k := make([]byte, size)

	k[0] = p                                        // Prefix Byte
	binary.BigEndian.PutUint64(k[1:9], uint64(cID)) // Context ID
	k[9] = order                                    // Order Byte

	for i, id := range currentIDs {
		binary.BigEndian.PutUint32(k[10+(i*4):14+(i*4)], id)
	}

	if nextID != nil {
		binary.BigEndian.PutUint32(k[size-4:], *nextID)
	}
	return k
}

func (s *PebbleStorage) addTransition(batch *pebble.Batch, cID int64, currentIDs []uint32, nextID uint32) error {
	val := make([]byte, 4)
	binary.BigEndian.PutUint32(val, 1)

	tKey := s.buildKey(prefixTransition, cID, currentIDs, &nextID)
	sKey := s.buildKey(prefixSum, cID, currentIDs, nil)

	err := batch.Merge(tKey, val, pebble.NoSync)
	if err != nil {
		return err
	}
	err = batch.Merge(sKey, val, pebble.NoSync)
	if err != nil {
		return err
	}
	return nil
}

func (s *PebbleStorage) GetWordID(word string) (uint32, error) {
	wordKey := append([]byte{prefixWordToID}, []byte(word)...)
	val, closer, err := s.db.Get(wordKey)
	if err != nil {
		return 0, err
	}
	defer closer.Close()
	return binary.BigEndian.Uint32(val), nil
}

func (s *PebbleStorage) GetOrCreateID(word string) (uint32, error) {
	id, err := s.GetWordID(word)
	if err == nil {
		return id, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if another goroutine created the ID while we were waiting for the lock.
	id, err = s.GetWordID(word)
	if err == nil {
		return id, nil
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	// Handle global counter for new IDs
	counterKey := []byte("sys_id_counter")
	var nextID uint32 = 1
	if v, closer, err := s.db.Get(counterKey); err == nil {
		nextID = binary.BigEndian.Uint32(v) + 1
		closer.Close()
	}

	idBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(idBytes, nextID)
	batch.Set(append([]byte{prefixWordToID}, []byte(word)...), idBytes, pebble.NoSync)
	batch.Set(append([]byte{prefixIDToWord}, idBytes...), []byte(word), pebble.NoSync)
	batch.Set(counterKey, idBytes, pebble.NoSync)

	return nextID, batch.Commit(pebble.Sync)
}

func (s *PebbleStorage) GetWord(id uint32) (string, error) {
	idBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(idBytes, id)
	val, closer, err := s.db.Get(append([]byte{prefixIDToWord}, idBytes...))
	if err != nil {
		return "", err
	}
	defer closer.Close()
	return string(val), nil
}

// GetOrCreateMultiple takes a list of words and ensures they all exist in the
// database, creating them in a single batch if necessary.
func (s *PebbleStorage) GetOrCreateMultiple(words []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newWords := make(map[string]struct{})
	for _, word := range words {
		wordKey := append([]byte{prefixWordToID}, []byte(word)...)
		if _, closer, err := s.db.Get(wordKey); err == pebble.ErrNotFound {
			newWords[word] = struct{}{}
		} else if err == nil {
			closer.Close()
		} else {
			return err
		}
	}

	if len(newWords) == 0 {
		return nil
	}

	batch := s.db.NewBatch()
	defer batch.Close()
	counterKey := []byte("sys_id_counter")
	var nextID uint32 = 1
	if v, closer, err := s.db.Get(counterKey); err == nil {
		nextID = binary.BigEndian.Uint32(v) + 1
		closer.Close()
	}

	for word := range newWords {
		idBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(idBytes, nextID)
		batch.Set(append([]byte{prefixWordToID}, []byte(word)...), idBytes, pebble.NoSync)
		batch.Set(append([]byte{prefixIDToWord}, idBytes...), []byte(word), pebble.NoSync)
		nextID++
	}

	idBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(idBytes, nextID-1)
	batch.Set(counterKey, idBytes, pebble.NoSync)

	return batch.Commit(pebble.Sync)
}

// ImportTransitions is a highly efficient, low-level function to bulk-import transitions.
// It uses a single database batch to write all data.
func (s *PebbleStorage) AddTransitions(transitions []Transition) error {
	batch := s.db.NewBatch()
	defer batch.Close()

	val := make([]byte, 4)

	for _, t := range transitions {
		if t.Frequency == 0 {
			continue
		}
		binary.BigEndian.PutUint32(val, t.Frequency)

		// Build the key for the transition itself ('t' prefix)
		tKey := s.buildKey(prefixTransition, t.CID, t.CurrentIDs, &t.NextID)
		if err := batch.Merge(tKey, val, pebble.NoSync); err != nil {
			return err
		}

		// Build the key for the N-gram's total frequency ('s' prefix)
		sKey := s.buildKey(prefixSum, t.CID, t.CurrentIDs, nil)
		if err := batch.Merge(sKey, val, pebble.NoSync); err != nil {
			return err
		}
	}

	return batch.Commit(pebble.Sync)
}

// Utility for Pebble prefix boundary
func upperBound(prefix []byte) []byte {
	res := make([]byte, len(prefix))
	copy(res, prefix)
	for i := len(res) - 1; i >= 0; i-- {
		res[i]++
		if res[i] != 0 {
			return res
		}
	}
	return nil
}
