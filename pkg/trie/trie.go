package trie

import (
	"fmt"
	"io"
	"math"
)

// convenience type for storing results
// also allows us to track the keys
// so that we can enumerate them at the end
type Trie struct {
	Root *Node
}

func NewTrie() *Trie {
	return &Trie{
		Root: makeNode(0),
	}
}

func (t *Trie) Insert(key []rune, val int) {
	t.Root.insert(key, val)
}

func (t Trie) GetNode(key []rune) *Node {
	curr := t.Root
	for _, r := range key {
		curr = curr.GetChild(r)
		if curr == nil {
			return nil
		}
	}

	return curr
}

func (t *Trie) Merge(other *Trie) {
	t.Root.merge(other.Root)
}

type Node struct {
	r        rune
	Children []*Node
	hasData  bool // if false, ignore fields below

	key   []rune
	Min   int
	Max   int
	Sum   int
	Count int
}

func makeNode(r rune) *Node {
	return &Node{
		r:        r,
		Children: make([]*Node, 0),
		hasData:  false,
		Min:      math.MaxInt,
		Max:      math.MinInt,
		Sum:      0,
		Count:    0,
	}
}

// returns the address of the node for the rune
// if it doesn't exist it returns the index to insert it at!
func (n Node) findChild(r rune) (int, bool) {
	var m int
	l, h := 0, len(n.Children)-1

	for l <= h {
		m = (l + h) / 2
		cr := n.Children[m].r
		if cr == r {
			return m, true
		}

		if cr < r {
			l = m + 1
		} else {
			h = m - 1
		}
	}

	return l, false
}

func (n *Node) GetChild(r rune) *Node {
	if i, found := n.findChild(r); found {
		return n.Children[i]
	} else {
		return nil
	}
}

func (n *Node) GetOrInsertChild(r rune) *Node {
	i, found := n.findChild(r)
	if found {
		return n.Children[i]
	}

	node := makeNode(r)
	node.key = append(n.key, r)
	n.Children = append(n.Children[:i], append([]*Node{node}, n.Children[i:]...)...)
	return node
}

func (n *Node) Update(val int) {
	n.hasData = true
	n.Max = max(n.Max, val)
	n.Min = min(n.Min, val)
	n.Sum += val
	n.Count++
}

func (n *Node) insert(key []rune, val int) {
	np := n
	for _, r := range key {
		np = np.GetOrInsertChild(r)
	}

	np.Update(val)
}

func (t Trie) Write(w io.Writer) {
	fmt.Fprint(w, "{")
	t.Root.Write(w, true)
	fmt.Fprintln(w, "}")
}

func (np Node) Write(w io.Writer, first bool) bool {
	if np.hasData {
		pfx := ", "
		if first {
			pfx = ""
			first = false
		}

		min := float64(np.Min) / 10.0
		max := float64(np.Max) / 10.0
		avg := math.Ceil(float64(np.Sum)/float64(np.Count)) / 10.0
		fmt.Fprintf(w, "%s%s=%.1f/%.1f/%.1f", pfx, string(np.key), min, avg, max)
	}

	wrote := np.hasData
	for _, child := range np.Children {
		first = first && !wrote
		wrote = child.Write(w, first) || wrote
	}

	return wrote
}

func (np *Node) merge(op *Node) {
	if np.hasData && op.hasData {
		np.Min = min(np.Min, op.Min)
		np.Max = max(np.Max, op.Max)
		np.Sum += op.Sum
		np.Count += op.Count
	}

	// they are both guaranteed to have sorted children so we can merge them in one pass
	i, j := 0, 0
	for i < len(np.Children) && j < len(op.Children) {
		cn := np.Children[i]
		co := op.Children[j]

		if cn.r == co.r {
			cn.merge(co)
			i++
			j++
		} else if cn.r < co.r {
			i++
		} else {
			np.Children = append(np.Children[:i], append([]*Node{co}, np.Children[i:]...)...)
			j++
		}
	}

	for ; j < len(op.Children); j++ {
		np.Children = append(np.Children, op.Children[j])
	}
}
