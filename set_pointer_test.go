package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestSet_PointersToScalars(t *testing.T) {
	type myString string
	type sample struct {
		Int     *int      `default:"1"`
		Uint    *uint     `default:"1"`
		Float32 *float32  `default:"1.32"`
		Bool    *bool     `default:"true"`
		String  *string   `default:"hello"`
		Named   *myString `default:"hello"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Int)
	require.NotNil(t, got.Uint)
	require.NotNil(t, got.Float32)
	require.NotNil(t, got.Bool)
	require.NotNil(t, got.String)
	require.NotNil(t, got.Named)

	assert.Equal(t, 1, *got.Int)
	assert.Equal(t, uint(1), *got.Uint)
	assert.Equal(t, float32(1.32), *got.Float32)
	assert.True(t, *got.Bool)
	assert.Equal(t, "hello", *got.String)
	assert.Equal(t, myString("hello"), *got.Named)
}

func TestSet_PointerToStruct(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Ptr *inner `default:"{}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Ptr)
	assert.Equal(t, inner{Name: "inner"}, *got.Ptr)
}

func TestSet_PointerToStructWithJSONTag(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Ptr *inner `default:"{\"Foo\": 123}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Ptr)
	assert.Equal(t, inner{Name: "inner", Foo: 123}, *got.Ptr, "the tag fills Foo, the nested tag fills Name")
}

func TestSet_PointerToMap(t *testing.T) {
	type sample struct {
		Empty *map[string]int `default:"{}"`
		JSON  *map[string]int `default:"{\"foo\": 123}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Empty)
	require.NotNil(t, got.JSON)
	assert.NotNil(t, *got.Empty, "the pointee is an allocated map, not a nil one")
	assert.Empty(t, *got.Empty)
	assert.Equal(t, map[string]int{"foo": 123}, *got.JSON)
}

func TestSet_PointerToSlice(t *testing.T) {
	type sample struct {
		Empty *[]string `default:"[]"`
		JSON  *[]string `default:"[\"foo\"]"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Empty)
	require.NotNil(t, got.JSON)
	assert.NotNil(t, *got.Empty, "the pointee is an allocated slice, not a nil one")
	assert.Empty(t, *got.Empty)
	assert.Equal(t, []string{"foo"}, *got.JSON)
}

func TestSet_UntaggedPointerStaysNil(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		Int    *int
		Struct *inner
		Map    *map[string]int
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Nil(t, got.Int)
	assert.Nil(t, got.Struct, "an untagged nil pointer is never allocated, not even to a struct")
	assert.Nil(t, got.Map)
}

// TestSet_UntaggedPointerStructRecurses covers the pointer a caller has already allocated: even
// with no tag, Set descends into it and fills its fields.
func TestSet_UntaggedPointerStructRecurses(t *testing.T) {
	type child struct {
		Name string `default:"Tom"`
		Age  int    `default:"20"`
	}
	type parent struct {
		Child *child
	}

	got := parent{Child: &child{Name: "Jim"}}
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Child)
	assert.Equal(t, child{Name: "Jim", Age: 20}, *got.Child, "the caller's name survives, the missing age is filled")
}

// TestSet_CallerPointerToContainerIsSkipped covers a pointer the caller allocated to a slice, a map or
// a pointer, as a field, a slice element and a map value.
//
// QUIRK: Set does not descend into such a pointer, so what a caller's *[]T, *map[K]T or **T holds
// gets no defaults, a default that would fail there is not reported, and a tag on the pointer stops
// at it. Yet a *[]T or *map[K]T held as a map value is descended into, and has been since v1.6.0
// (TestSet_MapOfPointerContainers), as a caller's pointer to a struct is anywhere
// (TestSet_UntaggedPointerStructRecurses). https://github.com/creasty/defaults/pull/100 descended
// into all three and was closed in favor of this pin: that walks every element behind such a
// pointer on each Set, and a pointer to a pointer can lead back to itself with no struct, slice or
// map between, which is all the cycle guard tracks.
func TestSet_CallerPointerToContainerIsSkipped(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}

	t.Run("a *[]T field", func(t *testing.T) {
		slice := []inner{{}}
		got := struct {
			Ptr *[]inner
		}{Ptr: &slice}

		require.NoError(t, defaults.Set(&got))

		assert.Empty(t, slice[0].Name)
	})

	t.Run("a *map[K]T field", func(t *testing.T) {
		mapping := map[string]inner{"a": {}}
		got := struct {
			Ptr *map[string]inner
		}{Ptr: &mapping}

		require.NoError(t, defaults.Set(&got))

		assert.Empty(t, mapping["a"].Name)
	})

	t.Run("a **T field", func(t *testing.T) {
		ptr := &inner{}
		got := struct {
			Ptr **inner
		}{Ptr: &ptr}

		require.NoError(t, defaults.Set(&got))

		assert.Empty(t, ptr.Name)
	})

	t.Run("a *[]T slice element", func(t *testing.T) {
		slice := []inner{{}}
		got := struct {
			Slices []*[]inner
		}{Slices: []*[]inner{&slice}}

		require.NoError(t, defaults.Set(&got))

		assert.Empty(t, slice[0].Name)
	})

	t.Run("a **T map value", func(t *testing.T) {
		ptr := &inner{}
		got := struct {
			Map map[string]**inner
		}{Map: map[string]**inner{"a": &ptr}}

		require.NoError(t, defaults.Set(&got))

		assert.Empty(t, ptr.Name)
	})

	t.Run("a tag on a *[]T field", func(t *testing.T) {
		var slice []string
		got := struct {
			Ptr *[]string `default:"[\"tag\"]"`
		}{Ptr: &slice}

		require.NoError(t, defaults.Set(&got))

		assert.Same(t, &slice, got.Ptr)
		assert.Nil(t, slice, "the tag does not reach the caller's nil slice")
	})

	t.Run("a default that would fail behind a *[]T field", func(t *testing.T) {
		type bad struct {
			Ints []int `default:"[!]"`
		}

		slice := []bad{{}}
		got := struct {
			Ptr *[]bad
		}{Ptr: &slice}

		require.NoError(t, defaults.Set(&got))

		assert.Nil(t, slice[0].Ints)
	})
}

// TestSet_PointerWithEmptyTag covers `default:""` on a pointer. The tag is present, so the pointer
// is allocated and points at the zero value — it used to stay nil, which left no way to ask for a
// pointer to an empty string. See https://github.com/creasty/defaults/issues/52.
//
// What makes this expressible is Set reading the tag with Tag.Lookup rather than Tag.Get, so an
// empty tag differs from no tag; TestSet_UntaggedPointerStaysNil covers the other side.
func TestSet_PointerWithEmptyTag(t *testing.T) {
	type sample struct {
		String *string `default:""`
		Int    *int    `default:""`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.String)
	require.NotNil(t, got.Int)
	assert.Empty(t, *got.String)
	assert.Zero(t, *got.Int)
}

// TestSet_NestedPointersAreAllocated covers arbitrary pointer depth: every level is allocated and
// the value lands at the bottom.
func TestSet_NestedPointersAreAllocated(t *testing.T) {
	type sample struct {
		Int    **int     `default:"1"`
		String ***string `default:"hello"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Int)
	require.NotNil(t, *got.Int)
	require.NotNil(t, got.String)
	require.NotNil(t, *got.String)
	require.NotNil(t, **got.String)

	assert.Equal(t, 1, **got.Int)
	assert.Equal(t, "hello", ***got.String)
}
