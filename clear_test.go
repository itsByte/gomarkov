package gomarkov

import (
	"os"
	"testing"
)

func newTestChain(t *testing.T, order int) (*Chain, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "gomarkov_test")
	if err != nil {
		t.Fatal(err)
	}
	storage, err := NewPebbleStorage(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatal(err)
	}
	chain := NewChain(order, storage)
	return chain, func() {
		storage.db.Close()
		os.RemoveAll(tempDir)
	}
}

func TestClearContext(t *testing.T) {
	chain, cleanup := newTestChain(t, 1)
	defer cleanup()

	chain.Add(1, []string{"hello", "world"})
	chain.Add(2, []string{"foo", "bar"})

	// Sanity: context 1 generates.
	words, _ := chain.GenerateAll(1)
	if len(words) == 0 {
		t.Fatal("expected generation before clear")
	}

	if err := chain.ClearContext(1); err != nil {
		t.Fatal(err)
	}

	// Context 1 is empty now.
	words, _ = chain.GenerateAll(1)
	if len(words) != 0 {
		t.Errorf("expected empty generation after clear, got %v", words)
	}

	// Context 2 still works.
	words, _ = chain.GenerateAll(2)
	if len(words) == 0 {
		t.Error("context 2 should be unaffected by clearing context 1")
	}
}

func TestClearContextTokens(t *testing.T) {
	chain, cleanup := newTestChain(t, 1)
	defer cleanup()

	// Two message types in the same context.
	chain.Add(1, []string{"\u001F_TEXT", "hello", "world"})
	chain.Add(1, []string{"\u001F_PHOTO", "abc123", "nice", "pic"})

	// Both types generate.
	words, _ := chain.GenerateAll(1)
	if len(words) == 0 {
		t.Fatal("expected generation before clear")
	}

	// Delete only text transitions.
	if err := chain.ClearContextTokens(1, []string{"\u001F_TEXT"}); err != nil {
		t.Fatal(err)
	}

	// Generation should still work (media remains) but must not start with text token.
	for range 10 {
		words, _ := chain.GenerateAll(1)
		if len(words) > 0 && words[0] == "\u001F_TEXT" {
			t.Fatalf("text token should be deleted, got %v", words)
		}
	}

	// Sums must be consistent: generation should not dead-end prematurely.
	words, _ = chain.GenerateAll(1)
	if len(words) == 0 {
		t.Error("expected non-empty generation from remaining media transitions")
	}
}
