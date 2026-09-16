package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// cycleCounter counts its setter's calls, so a test can tell how many times the walk reached it. It
// is package-level only because it needs a method.
type cycleCounter struct {
	Calls int
}

func (c *cycleCounter) SetDefaults() {
	c.Calls++
}

// TestSet_CyclesEnd covers data the caller built with a way back to itself. Where the walk comes back
// to a value it is still walking, Set goes no further and returns, and the walk already under way
// finishes that value. It used to follow the cycle until the stack overflowed: a fatal error, which
// recover cannot catch, so it took the whole process down.
//
// A tag that creates another of itself is a different matter, and an error; that is
// TestSet_RejectsDefaultThatRecursesWithoutEnd.
func TestSet_CyclesEnd(t *testing.T) {
	t.Run("a pointer back to the parent", func(t *testing.T) {
		type node struct {
			Name     string `default:"node"`
			Parent   *node
			Children []*node
		}

		root := &node{}
		child := &node{Parent: root}
		root.Children = []*node{child}

		require.NoError(t, defaults.Set(root))

		assert.Equal(t, "node", root.Name)
		assert.Equal(t, "node", child.Name)
		assert.Same(t, root, child.Parent, "the cycle is kept as the caller built it")
	})

	t.Run("a pointer to itself", func(t *testing.T) {
		type node struct {
			Name string `default:"node"`
			Next *node
		}

		n := &node{}
		n.Next = n

		require.NoError(t, defaults.Set(n))

		assert.Equal(t, "node", n.Name)
		assert.Same(t, n, n.Next)
	})

	// Set is handed the root as a named pointer type, and the cycle leads back to it as a plain
	// *node. Taken for another value, the root would be walked a second time, reaching Counter again.
	t.Run("a named pointer type back to the root", func(t *testing.T) {
		type node struct {
			Name    string `default:"node"`
			Next    *node
			Counter cycleCounter
		}
		type nodePtr *node

		n := &node{}
		n.Next = n

		require.NoError(t, defaults.Set(nodePtr(n)))

		assert.Equal(t, "node", n.Name)
		assert.Equal(t, 1, n.Counter.Calls, "the root is walked once")
	})

	t.Run("a slice holding itself with no struct between", func(t *testing.T) {
		type loop []loop

		l := loop{nil}
		l[0] = l
		got := struct {
			Loop loop
		}{Loop: l}

		require.NoError(t, defaults.Set(&got))

		assert.Len(t, got.Loop, 1)
	})

	t.Run("a slice holding its own array", func(t *testing.T) {
		type node struct {
			Name     string `default:"node"`
			Children []node
		}

		nodes := make([]node, 1)
		nodes[0].Children = nodes

		require.NoError(t, defaults.Set(&nodes[0]))

		assert.Equal(t, "node", nodes[0].Name)
	})

	// The longer slice starts where the shorter one does, with the same type, so only its length
	// tells it apart; taken for the shorter one, it would be skipped, and nodes[2] never filled.
	t.Run("a longer slice over an array already on the path", func(t *testing.T) {
		type node struct {
			Name     string `default:"node"`
			Children []node
		}

		nodes := make([]node, 3)
		nodes[1].Children = nodes
		got := struct {
			Nodes []node
		}{Nodes: nodes[:2]}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "node", nodes[0].Name)
		assert.Equal(t, "node", nodes[1].Name)
		assert.Equal(t, "node", nodes[2].Name, "reached only through the longer slice")
	})

	t.Run("a map holding itself", func(t *testing.T) {
		type node struct {
			Name     string `default:"node"`
			Children map[string]node
		}

		children := map[string]node{}
		children["self"] = node{Children: children}
		got := node{Children: children}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "node", got.Name)
		assert.Equal(t, "node", children["self"].Name)
	})
}

// TestSet_CyclesEndAtAnyDepth covers cycles that close far down a long path. Set scans the path for a
// value near its top, but looks up a value further down in an index of the path instead, so the index
// has to hold every value above, those near the top as well. Each link's counter shows it is walked
// once, however deep the value the cycle leads back to.
func TestSet_CyclesEndAtAnyDepth(t *testing.T) {
	const links = 1000

	tests := []struct {
		name string
		back int
	}{
		{"the last link back to the first", 0},
		{"the last link back to one halfway down", links / 2},
		{"the last link back to itself", links - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type node struct {
				Name    string `default:"node"`
				Next    *node
				Counter cycleCounter
			}

			nodes := make([]*node, links)
			for i := range nodes {
				nodes[i] = &node{}
				if i > 0 {
					nodes[i-1].Next = nodes[i]
				}
			}
			nodes[links-1].Next = nodes[tt.back]

			require.NoError(t, defaults.Set(nodes[0]))

			for i, n := range nodes {
				require.Equal(t, "node", n.Name, "link %d", i)
				require.Equal(t, 1, n.Counter.Calls, "link %d is walked once", i)
			}
			assert.Same(t, nodes[tt.back], nodes[links-1].Next, "the cycle is kept as the caller built it")
		})
	}

	// The path to the links passes through a slice, which counts toward their depth as a struct does,
	// so the struct at the top, above the slice, is in the index too when the last link leads back to
	// it.
	t.Run("links below a slice, the last back to the top", func(t *testing.T) {
		type node struct {
			Next    *node
			Items   []node
			Counter cycleCounter
		}

		top := &node{Items: make([]node, 1)}
		last := &top.Items[0]
		for i := 0; i < links; i++ {
			last.Next = &node{}
			last = last.Next
		}
		last.Next = top

		require.NoError(t, defaults.Set(top))

		assert.Equal(t, 1, top.Counter.Calls, "the top is walked once")
		assert.Equal(t, 1, top.Items[0].Counter.Calls, "the slice's element is walked once")
		assert.Equal(t, 1, last.Counter.Calls, "the last link is walked once")
	})

	// The index made, and grown, for the first chain is still the path's when the second chain is
	// walked, and the second's last link leads back to the struct at the top: taking the first chain's
	// links out has to leave that struct in. Each round's structs have new addresses, so the index's
	// buckets fall differently each time.
	t.Run("a second chain as long, the last link back to the top", func(t *testing.T) {
		type node struct {
			First   *node
			Second  *node
			Counter cycleCounter
		}

		chain := func() (head, last *node) {
			head = &node{}
			last = head
			for i := 1; i < links; i++ {
				last.First = &node{}
				last = last.First
			}
			return head, last
		}

		for round := 0; round < 5; round++ {
			top := &node{}
			top.First, _ = chain()
			var last *node
			top.Second, last = chain()
			last.First = top

			require.NoError(t, defaults.Set(top))

			require.Equal(t, 1, top.Counter.Calls, "round %d: the top is walked once", round)
			for i, n := 0, top.Second; n != top; i, n = i+1, n.First {
				require.Equal(t, 1, n.Counter.Calls, "round %d: link %d of the second chain", round, i)
			}
		}
	})

	// The longer slice of TestSet_CyclesEnd, at the bottom of the links: it is told apart from the
	// shorter one by its length in the index too.
	t.Run("a longer slice over an array already on the path", func(t *testing.T) {
		type node struct {
			Name     string `default:"node"`
			Next     *node
			Children []node
		}

		nodes := make([]node, 3)
		nodes[1].Children = nodes
		top := &node{Children: nodes[:2]}
		for i := 0; i < links; i++ {
			top = &node{Next: top}
		}

		require.NoError(t, defaults.Set(top))

		assert.Equal(t, "node", nodes[0].Name)
		assert.Equal(t, "node", nodes[1].Name)
		assert.Equal(t, "node", nodes[2].Name, "reached only through the longer slice")
	})
}
