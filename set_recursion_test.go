package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// TestSet_RejectsDefaultThatRecursesWithoutEnd covers a type that refers to itself through a field
// whose tag creates what it refers to: the default fills a zero value that holds the same field, so
// it is applied again below, and again below that. Set returns an error naming the field.
//
// It used to recurse until the stack overflowed, which is a fatal error rather than a panic: recover
// could not catch it, and a test of it took the whole test binary down. See
// https://github.com/creasty/defaults/issues/71.
func TestSet_RejectsDefaultThatRecursesWithoutEnd(t *testing.T) {
	type ptrNode struct {
		Next *ptrNode `default:"{}"`
	}
	type sliceNode struct {
		Children []sliceNode `default:"[{}]"`
	}
	type mapNode struct {
		Children map[string]mapNode `default:"{\"a\": {}}"`
	}
	// An empty tag still allocates a pointer, so it recurses like any other.
	type emptyTagNode struct {
		Next *emptyTagNode `default:""`
	}
	// The cycle can pass through another type. This one is anonymous only because a type declared
	// in a function cannot refer to one declared after it.
	type viaAnother struct {
		Inner *struct {
			Outer *viaAnother `default:"{}"`
		} `default:"{}"`
	}
	// Or through no struct at all: a pointer's tag carries on to what it points at, which here is
	// the same pointer type again.
	type ptrLoop *ptrLoop
	type ptrLoopHolder struct {
		Loop ptrLoop `default:"x"`
	}

	tests := []struct {
		name string
		ptr  interface{}
		want string
	}{
		{"pointer", &ptrNode{}, `field Next: default "{}" recurses without end`},
		{"slice", &sliceNode{}, `field Children: default "[{}]" recurses without end`},
		{"map", &mapNode{}, `field Children: default "{\"a\": {}}" recurses without end`},
		{"empty tag on a pointer", &emptyTagNode{}, `field Next: default "" recurses without end`},
		{"through another type", &viaAnother{}, `field Inner: default "{}" recurses without end`},
		{"through a pointer to itself", &ptrLoopHolder{}, `field Loop: default "x" recurses without end`},
		// However long the chain, it ends in a nil the tag fills.
		{"chain the caller built", &ptrNode{Next: &ptrNode{Next: &ptrNode{}}}, `field Next: default "{}" recurses without end`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, defaults.Set(tt.ptr), tt.want)
		})
	}
}

// TestSet_RecursiveTypeEnds covers types that repeat without their default recursing, which Set must
// walk to the end rather than reject. A repeat counts only when a tag meets a zero value of the type
// below where it is already filling one, and each case misses that one way: no tag, a tag that
// creates nothing, a value already filled in, a different tag, or the same tag beside rather than
// below.
func TestSet_RecursiveTypeEnds(t *testing.T) {
	t.Run("an untagged pointer", func(t *testing.T) {
		type node struct {
			Name string `default:"node"`
			Next *node
		}

		got := node{Next: &node{}}
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, node{Name: "node", Next: &node{Name: "node"}}, got,
			"the caller's chain is walked to its end, and the nil there stays nil")
	})

	t.Run("an empty slice tag", func(t *testing.T) {
		type node struct {
			Children []node `default:"[]"`
		}

		var got node
		require.NoError(t, defaults.Set(&got))

		assert.NotNil(t, got.Children)
		assert.Empty(t, got.Children, "there is no element to descend into")
	})

	t.Run("leaves the caller allocated", func(t *testing.T) {
		type node struct {
			Name     string `default:"node"`
			Children []node `default:"[{}]"`
		}

		got := node{Children: []node{{Children: []node{}}}}
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, node{Name: "node", Children: []node{{Name: "node", Children: []node{}}}}, got,
			"an allocated Children is not zero even when empty, so the tag skips it")
	})

	t.Run("leaves the tag allocates", func(t *testing.T) {
		type node struct {
			Name     string `default:"node"`
			Children []node `default:"[{\"Children\": []}]"`
		}

		var got node
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, node{Name: "node", Children: []node{{Name: "node", Children: []node{}}}}, got,
			"the element's Children is allocated by the same tag, so the tag does not apply to it again")
	})

	t.Run("the same type under a different tag", func(t *testing.T) {
		type node struct {
			First []node `default:"[{\"First\": []}]"`
			Rest  []node `default:"[]"`
		}

		var got node
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, node{First: []node{{First: []node{}, Rest: []node{}}}, Rest: []node{}}, got,
			"a zero []node is filled below another, but by a tag that creates no element")
	})

	t.Run("the same type and tag side by side", func(t *testing.T) {
		type leaf struct {
			Name string `default:"leaf"`
		}
		type pair struct {
			Left  *leaf `default:"{}"`
			Right *leaf `default:"{}"`
		}

		var got pair
		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Left)
		require.NotNil(t, got.Right)
		assert.Equal(t, "leaf", got.Left.Name)
		assert.Equal(t, "leaf", got.Right.Name, "a sibling is not below the other, so it is no repeat")
	})
}
