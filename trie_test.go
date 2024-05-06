package main

import (
	"os"
	"testing"
)

func TestTrie(t *testing.T) {
	entries := []struct {
		key string
		val int
	}{
		{"Sacramento", 1},
		{"San Francisco", 2},
		{"San Jose", 3},
	}

	trie := NewTrie()
	for _, entry := range entries {
		trie.Insert([]byte(entry.key), entry.val)
	}

	trie.Output(os.Stdout)
}
