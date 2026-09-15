package defaults_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// aliasCounter counts its setter's calls, so a test can tell a value filled once from one filled
// once per path to it. It is package-level only because it needs a method.
type aliasCounter struct {
	Name  string `default:"counter"`
	Calls int
}

func (c *aliasCounter) SetDefaults() {
	c.Calls++
}

// TestSet_CyclesEnd covers data the caller built with a way back to itself. Set fills each value
// once and returns, where it used to follow the cycle until the stack overflowed: a fatal error,
// which recover cannot catch, so it took the whole process down.
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

// TestSet_SharedValuesAreFilledOnce covers a value the caller made reachable by more than one path:
// it is filled, and its SetDefaults called, once per Set. It used to be walked once per path, so a
// setter that was not idempotent applied itself once per path, and a chain of values each shared by
// two pointers took time exponential in its length.
func TestSet_SharedValuesAreFilledOnce(t *testing.T) {
	t.Run("two pointers", func(t *testing.T) {
		shared := &aliasCounter{}
		got := struct {
			First  *aliasCounter
			Second *aliasCounter
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, shared.Calls)
	})

	t.Run("a pointer to a sibling field", func(t *testing.T) {
		got := struct {
			Value aliasCounter
			Ptr   *aliasCounter
		}{}
		got.Ptr = &got.Value

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Value.Calls)
	})

	t.Run("two slices over one array", func(t *testing.T) {
		shared := []aliasCounter{{}}
		got := struct {
			First  []aliasCounter
			Second []aliasCounter
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, shared[0].Calls)
	})

	t.Run("two fields holding one map", func(t *testing.T) {
		shared := map[string]aliasCounter{"a": {}}
		got := struct {
			First  map[string]aliasCounter
			Second map[string]aliasCounter
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, shared["a"].Calls)
	})

	t.Run("one map held under two types", func(t *testing.T) {
		type named map[string]aliasCounter

		shared := map[string]aliasCounter{"a": {}}
		got := struct {
			First  map[string]aliasCounter
			Second named
		}{First: shared, Second: named(shared)}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, shared["a"].Calls)
	})

	t.Run("a chain of values each shared by two pointers", func(t *testing.T) {
		type node struct {
			Left  *node
			Right *node
			Leaf  *aliasCounter
		}

		leaf := &aliasCounter{}
		n := &node{Leaf: leaf}
		for i := 0; i < 64; i++ {
			n = &node{Left: n, Right: n}
		}

		require.NoError(t, defaults.Set(n))

		assert.Equal(t, 1, leaf.Calls, "reached by 2^64 paths")
	})
}

// TestSet_ALaterPathStillAppliesItsTag covers a value reached first by a path whose tags do nothing
// to it, and then by one whose tags do: a pointer whose tag carries on to a struct left zero, or a
// map held again under an element type with tags. Being entered already keeps neither the tags from
// applying nor what they create, down to a struct inside the one decoded into, from being filled.
func TestSet_ALaterPathStillAppliesItsTag(t *testing.T) {
	type leaf struct {
		Name string `default:"leaf"`
	}
	type inner struct {
		Leaves []leaf
	}

	t.Run("after an untagged pointer to the struct", func(t *testing.T) {
		shared := &inner{}
		got := struct {
			First  *inner
			Second *inner `default:"{\"Leaves\": [{}]}"`
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, []leaf{{Name: "leaf"}}, shared.Leaves)
	})

	t.Run("into a struct inside the one decoded into", func(t *testing.T) {
		type outer struct {
			Mid inner
		}

		shared := &outer{}
		got := struct {
			First  *outer
			Second *outer `default:"{\"Mid\": {\"Leaves\": [{}]}}"`
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, []leaf{{Name: "leaf"}}, shared.Mid.Leaves)
	})

	t.Run("into a struct embedded in the one decoded into", func(t *testing.T) {
		type Embedded struct {
			Leaves []leaf
		}
		type outer struct {
			Embedded
		}

		shared := &outer{}
		got := struct {
			First  *outer
			Second *outer `default:"{\"Leaves\": [{}]}"`
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, []leaf{{Name: "leaf"}}, shared.Leaves)
	})

	t.Run("into a slice its own tag left nil", func(t *testing.T) {
		type holder struct {
			Leaves []leaf `default:"null"`
		}

		shared := &holder{}
		got := struct {
			First  *holder
			Second *holder `default:"{\"Leaves\": [{}]}"`
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, []leaf{{Name: "leaf"}}, shared.Leaves)
	})

	t.Run("into a struct holding an unexported struct", func(t *testing.T) {
		type outer struct {
			hidden inner
			Leaves []leaf
		}

		shared := &outer{}
		got := struct {
			First  *outer
			Second *outer `default:"{\"Leaves\": [{}]}"`
		}{First: shared, Second: shared}

		var err error
		require.NotPanics(t, func() {
			err = defaults.Set(&got)
		})

		require.NoError(t, err)
		assert.Equal(t, []leaf{{Name: "leaf"}}, shared.Leaves)
		assert.Empty(t, shared.hidden.Leaves, "the unexported struct is skipped, as it is anywhere")
	})

	t.Run("with a shared value entered after the struct", func(t *testing.T) {
		shared := &inner{}
		counter := &aliasCounter{}
		got := struct {
			First   *inner
			Counter *aliasCounter
			Second  *inner `default:"{\"Leaves\": [{}]}"`
			Again   *aliasCounter
		}{First: shared, Counter: counter, Second: shared, Again: counter}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, []leaf{{Name: "leaf"}}, shared.Leaves)
		assert.Equal(t, 1, counter.Calls, "forgetting the struct keeps what was entered after it")
	})

	t.Run("after the same map under an element type without tags", func(t *testing.T) {
		shared := map[string]struct{ Name string }{"a": {}}
		got := struct {
			First  map[string]struct{ Name string }
			Second map[string]struct {
				Name string `default:"leaf"`
			}
		}{First: shared}
		got.Second = map[string]struct {
			Name string `default:"leaf"`
		}(shared)

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "leaf", shared["a"].Name)
	})
}

// aliasHook runs a function when Set calls its setter, so a test can change the value partway
// through the walk. It is package-level only because it needs a method.
type aliasHook struct {
	run func()
}

func (h *aliasHook) SetDefaults() {
	h.run()
}

// TestSet_DroppedMapIsNotMistakenForANewOne covers a setter that drops a map Set has already walked,
// and builds new ones. The walk knows a map by the address of its table, and a table nothing refers
// to can be collected and its address given to a new map; the walk keeps the maps it has entered
// alive, so none is taken for a new one. Without that, a new map landed at the old address within a
// few hundred allocations in nearly every run, and was skipped. Five runs make a miss unlikely.
func TestSet_DroppedMapIsNotMistakenForANewOne(t *testing.T) {
	type leaf struct {
		Name string `default:"leaf"`
	}

	for run := 0; run < 5; run++ {
		got := struct {
			Old  map[string]*leaf
			Hook aliasHook
			New  []map[string]*leaf
		}{Old: map[string]*leaf{"a": {}}}
		got.Hook.run = func() {
			got.Old = nil
			runtime.GC()
			runtime.GC()
			for i := 0; i < 256; i++ {
				got.New = append(got.New, map[string]*leaf{"a": {}})
			}
		}

		require.NoError(t, defaults.Set(&got))

		for i, m := range got.New {
			require.Equal(t, "leaf", m["a"].Name, "run %d, map %d", run, i)
		}
	}
}
