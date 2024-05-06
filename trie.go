package main

import (
	"fmt"
	"io"
	"sort"
)

// convience type for storing results
// also allows us to track the keys
// so that we can enumerate them at the end
type Trie struct {
	Root *Node
	Keys map[string]struct{}
}

func NewTrie() *Trie {
	return &Trie{
		Root: makeNode(0),
		Keys: make(map[string]struct{}),
	}
}

func (t *Trie) Insert(key []byte, val int) {
	t.Root.insert(key, val)
	t.Keys[string(key)] = struct{}{}
}

func (t Trie) GetNode(key []byte) *Node {
	curr := t.Root
	for _, b := range key {
		curr = curr.getChild(b)
		if curr == nil {
			return nil
		}
	}

	return curr
}

func (t Trie) Output(w io.Writer) {
	w.Write([]byte{'{'})
	var prev *Node
	var prevString string
	for key := range t.Keys {
		if prev != nil {
			prev.output(prevString, w)
			w.Write([]byte{','})
		}

		n := t.GetNode([]byte(key))
		prev = n
		prevString = key
	}
	prev.output(prevString, w)
	w.Write([]byte{'}'})
}

type Node struct {
	b        byte
	Children []*Node
	Min      int
	Max      int
	Sum      int
	Count    int
}

func (n Node) output(key string, w io.Writer) {
	min := float64(n.Min) / 10.0
	max := float64(n.Max) / 10.0
	avg := float64(n.Sum) / float64(n.Count) / 10.0
	s := fmt.Sprintf("%s=%.1f/%.1f/%.1f, ", key, min, avg, max)
	w.Write([]byte(s))
}

func makeNode(b byte) *Node {
	return &Node{
		b:        b,
		Children: make([]*Node, 0),
	}
}

// returns the address of the node for the byte
func (n Node) findChild(b byte) (int, bool) {
	return sort.Find(len(n.Children), func(i int) int {
		return int(n.Children[i].b - b)
	})
}

func (n *Node) getChild(b byte) *Node {
	if i, found := n.findChild(b); found {
		return n.Children[i]
	} else {
		return nil
	}
}

func (n *Node) getOrInsertChild(b byte) *Node {
	fmt.Printf("Inserting b=%s for node=%s\n", string(b), string(n.b))
	i, found := n.findChild(b)
	fmt.Printf("\tfound=%t\n", found)
	if found {
		return n.Children[i]
	}

	node := makeNode(b)
	n.Children = append(n.Children[:i], append([]*Node{node}, n.Children[i:]...)...)
	return node
}

func (n *Node) update(val int) {
	if val > n.Max {
		n.Max = val
	}
	if val < n.Min {
		n.Min = val
	}
	n.Sum += val
	n.Count++
}

func (n *Node) insert(key []byte, val int) {
	curr := n
	for _, b := range key {
		curr = curr.getOrInsertChild(b)
	}
	curr.update(val)
}
