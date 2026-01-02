package gomarkov

import (
	"os"
	"reflect"
	"testing"
)

func TestChain_Add(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gomarkov_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewPebbleStorage(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.db.Close()

	chain := NewChain(2, storage)
	err = chain.Add(1, []string{"I", "like", "to", "eat", "cake"})
	if err != nil {
		t.Fatal(err)
	}
	err = chain.Add(1, []string{"I", "like", "to", "eat", "pizza"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestChain_Generate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gomarkov_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewPebbleStorage(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.db.Close()

	chain := NewChain(2, storage)
	err = chain.Add(1, []string{"I", "like", "to", "eat", "cake"})
	if err != nil {
		t.Fatal(err)
	}
	err = chain.Add(1, []string{"I", "like", "to", "eat", "pizza"})
	if err != nil {
		t.Fatal(err)
	}

	word, err := chain.Generate(1, []string{"I", "like"})
	if err != nil {
		t.Fatal(err)
	}
	if word != "to" {
		t.Errorf("Expected 'to', got %s", word)
	}
}

func TestChain_GenerateAll(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gomarkov_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewPebbleStorage(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.db.Close()

	chain := NewChain(1, storage)
	err = chain.Add(1, []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	words, err := chain.GenerateAll(1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(words, []string{"a", "b", "c"}) {
		t.Errorf("Expected [a b c], got %v", words)
	}
}
