package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// sharedCounter counts its setter's calls, so a test can tell how many paths reached it. It is
// package-level only because it needs a method.
type sharedCounter struct {
	Name  string `default:"counter"`
	Calls int
}

func (c *sharedCounter) SetDefaults() {
	c.Calls++
}

// TestSet_SharedValueIsFilledOncePerPath covers a value the caller made reachable by more than one
// path. Set knows only the values on the path it is walking, so a value two paths reach without one
// passing through the other is walked on each: its fields are looked at again, and its SetDefaults
// is called again.
//
// QUIRK: a SetDefaults that is not idempotent applies itself once per path. Calling it once takes a
// record of every value walked, which https://github.com/creasty/defaults/pull/99 kept: 256 B in two
// allocations for any Set that entered a value, and more past the eighth. Knowing only the path
// allocates nothing and still ends a cycle, so filling once per path stays, as the README says.
func TestSet_SharedValueIsFilledOncePerPath(t *testing.T) {
	t.Run("two pointers", func(t *testing.T) {
		shared := &sharedCounter{}
		got := struct {
			First  *sharedCounter
			Second *sharedCounter
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "counter", shared.Name)
		assert.Equal(t, 2, shared.Calls)
	})

	// The field and the pointer share an address, but the pointer is reached from the struct, not
	// from the field, so the field is not on its path.
	t.Run("a pointer to a sibling field", func(t *testing.T) {
		got := struct {
			Value sharedCounter
			Ptr   *sharedCounter
		}{}
		got.Ptr = &got.Value

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 2, got.Value.Calls)
	})

	t.Run("two slices over one array", func(t *testing.T) {
		shared := []sharedCounter{{}}
		got := struct {
			First  []sharedCounter
			Second []sharedCounter
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 2, shared[0].Calls)
	})

	t.Run("one map in two fields", func(t *testing.T) {
		shared := map[string]sharedCounter{"a": {}}
		got := struct {
			First  map[string]sharedCounter
			Second map[string]sharedCounter
		}{First: shared, Second: shared}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 2, shared["a"].Calls)
	})

	// QUIRK: each link doubles the paths to the leaf, and the calls with them, so the time a chain
	// takes doubles with every link too: 64 links would not finish. Only a record of every value
	// walked reaches the leaf once, and that is the cost above, which
	// https://github.com/creasty/defaults/pull/99 paid and this walk does not.
	t.Run("a chain of values each shared by two pointers", func(t *testing.T) {
		type node struct {
			Left  *node
			Right *node
			Leaf  *sharedCounter
		}

		for links := 0; links <= 4; links++ {
			leaf := &sharedCounter{}
			n := &node{Leaf: leaf}
			for i := 0; i < links; i++ {
				n = &node{Left: n, Right: n}
			}

			require.NoError(t, defaults.Set(n))

			assert.Equal(t, 1<<links, leaf.Calls, "%d links", links)
		}
	})
}

// TestSet_SharedValueIsFilledOncePerPathAtAnyDepth covers values shared far down a long path, where Set
// looks a value up in an index of the path instead of scanning the path for it. A value leaves the
// index as its walk returns, so a second path reaches it again, however deep either path reaches it.
func TestSet_SharedValueIsFilledOncePerPathAtAnyDepth(t *testing.T) {
	const links = 1000

	// The first path enters the shared chain at the top, where Set scans, and goes on far below it. The
	// second enters it 1000 links down, so every link of the chain is looked up there: any that the
	// first path left in the index would be taken for a repeat.
	t.Run("a chain reached again further down", func(t *testing.T) {
		type node struct {
			Next    *node
			Counter sharedCounter
		}

		var shared *node
		for i := 0; i < links; i++ {
			shared = &node{Next: shared}
		}
		detour := shared
		for i := 0; i < links; i++ {
			detour = &node{Next: detour}
		}
		got := struct {
			Direct *node
			Detour *node
		}{Direct: shared, Detour: detour}

		require.NoError(t, defaults.Set(&got))

		n := detour
		for i := 0; i < links; i++ {
			require.Equal(t, 1, n.Counter.Calls, "detour link %d", i)
			n = n.Next
		}
		for i := 0; n != nil; i++ {
			require.Equal(t, 2, n.Counter.Calls, "shared link %d", i)
			n = n.Next
		}
	})

	// The field shares the node's address, so only its type tells the two apart in the index.
	t.Run("a pointer to a sibling field", func(t *testing.T) {
		type node struct {
			Value sharedCounter
			Ptr   *sharedCounter
			Next  *node
		}

		bottom := &node{}
		bottom.Ptr = &bottom.Value
		top := bottom
		for i := 0; i < links; i++ {
			top = &node{Next: top}
		}

		require.NoError(t, defaults.Set(top))

		assert.Equal(t, 2, bottom.Value.Calls)
	})

	t.Run("two slices over one array", func(t *testing.T) {
		type node struct {
			Next   *node
			First  []sharedCounter
			Second []sharedCounter
		}

		shared := []sharedCounter{{}}
		top := &node{First: shared, Second: shared}
		for i := 0; i < links; i++ {
			top = &node{Next: top}
		}

		require.NoError(t, defaults.Set(top))

		assert.Equal(t, 2, shared[0].Calls)
	})

	t.Run("one map in two fields", func(t *testing.T) {
		type node struct {
			Next   *node
			First  map[string]sharedCounter
			Second map[string]sharedCounter
		}

		shared := map[string]sharedCounter{"a": {}}
		top := &node{First: shared, Second: shared}
		for i := 0; i < links; i++ {
			top = &node{Next: top}
		}

		require.NoError(t, defaults.Set(top))

		assert.Equal(t, 2, shared["a"].Calls)
	})
}
