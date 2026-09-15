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
