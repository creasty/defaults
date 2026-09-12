package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestSet_EmptyMapTagCreatesEmptyMap(t *testing.T) {
	type sample struct {
		Map map[string]int `default:"{}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.NotNil(t, got.Map, "an allocated map, not a nil one")
	assert.Empty(t, got.Map)
}

func TestSet_MapTagJSON(t *testing.T) {
	type sample struct {
		Map map[string]int `default:"{\"foo\": 123}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, map[string]int{"foo": 123}, got.Map)
}

func TestSet_NamedMapType(t *testing.T) {
	type myMap map[string]int
	type sample struct {
		Empty myMap `default:"{}"`
		JSON  myMap `default:"{\"foo\": 123}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.NotNil(t, got.Empty)
	assert.Empty(t, got.Empty)
	assert.Equal(t, myMap{"foo": 123}, got.JSON)
}

func TestSet_UntaggedMapStaysNil(t *testing.T) {
	type sample struct {
		Map map[string]int
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Nil(t, got.Map)
}

// TestSet_MapOfStructsGetsDefaults covers a map of struct values the caller filled in. Values are
// not addressable, so each element is copied, defaulted, and written back with SetMapIndex.
func TestSet_MapOfStructsGetsDefaults(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Map map[string]inner
	}

	got := sample{Map: map[string]inner{
		"a": {Foo: 1},
		"b": {Name: "kept"},
	}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, map[string]inner{
		"a": {Name: "inner", Foo: 1},
		"b": {Name: "kept"},
	}, got.Map, "the caller's entries survive and gain their missing defaults")
}

func TestSet_MapOfPointerStructsGetsDefaults(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Map map[string]*inner
	}

	got := sample{Map: map[string]*inner{
		"a": {Foo: 1},
		"b": {},
	}}
	require.NoError(t, defaults.Set(&got))

	require.Len(t, got.Map, 2)
	require.NotNil(t, got.Map["a"])
	require.NotNil(t, got.Map["b"])
	assert.Equal(t, inner{Name: "inner", Foo: 1}, *got.Map["a"])
	assert.Equal(t, inner{Name: "inner"}, *got.Map["b"])
}

// TestSet_MapOfNilPointersIsLeftAlone covers a nil pointer as a map value: there is nothing behind
// it to recurse into. A pointer map value is written through rather than copied back, so allocating
// one here would have to be a deliberate choice, and it is not made -- unlike a nil pointer *field*,
// which a tag does allocate.
func TestSet_MapOfNilPointersIsLeftAlone(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}

	t.Run("from the caller", func(t *testing.T) {
		got := struct {
			Map map[string]*inner
		}{Map: map[string]*inner{"a": nil}}

		require.NoError(t, defaults.Set(&got))

		require.Contains(t, got.Map, "a", "the key stays")
		assert.Nil(t, got.Map["a"], "with no struct behind it")
	})

	t.Run("from the tag", func(t *testing.T) {
		got := struct {
			Map map[string]*inner `default:"{\"a\": null}"`
		}{}

		require.NoError(t, defaults.Set(&got))

		require.Contains(t, got.Map, "a")
		assert.Nil(t, got.Map["a"])
	})
}

// TestSet_MapTagCreatesStructElements covers elements the tag itself creates: they are defaulted
// like any other element, and no other key appears.
func TestSet_MapTagCreatesStructElements(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Map map[string]inner `default:"{\"a\": {\"Foo\": 123}}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.Len(t, got.Map, 1)
	assert.NotContains(t, got.Map, "b")
	assert.Equal(t, inner{Name: "inner", Foo: 123}, got.Map["a"])
}

func TestSet_MapTagCreatesPointerStructElements(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Map map[string]*inner `default:"{\"a\": {\"Foo\": 123}}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.Len(t, got.Map, 1)
	require.NotNil(t, got.Map["a"])
	assert.Equal(t, inner{Name: "inner", Foo: 123}, *got.Map["a"])
}

// TestSet_MapOfScalarsIsLeftAlone pins that scalar map values are never rewritten: a zero value is
// not a field with a tag.
func TestSet_MapOfScalarsIsLeftAlone(t *testing.T) {
	type sample struct {
		Map map[string]int `default:"{\"zero\": 0, \"one\": 1}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, map[string]int{"zero": 0, "one": 1}, got.Map)
}

// TestSet_MapOfPointerContainers covers map values that point at a container rather than a struct.
// The element loop handles all three of struct, slice and map behind a pointer.
func TestSet_MapOfPointerContainers(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		Slices map[string]*[]inner
		Maps   map[string]*map[string]inner
	}

	slice := []inner{{}}
	nested := map[string]inner{"x": {}}
	got := sample{
		Slices: map[string]*[]inner{"a": &slice},
		Maps:   map[string]*map[string]inner{"a": &nested},
	}
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Slices["a"])
	require.Len(t, *got.Slices["a"], 1)
	assert.Equal(t, "inner", (*got.Slices["a"])[0].Name)

	require.NotNil(t, got.Maps["a"])
	assert.Equal(t, "inner", (*got.Maps["a"])["x"].Name)
}

// TestSet_MapTagIsNotReappliedToElements pins that the element recursion is handed an empty tag: the
// parent's JSON builds the map once and is not decoded again into each element. The values here are
// chosen so that re-applying it would be an error rather than a no-op.
func TestSet_MapTagIsNotReappliedToElements(t *testing.T) {
	t.Run("element is a container", func(t *testing.T) {
		got := struct {
			Map map[string][]int `default:"{\"a\": null}"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, map[string][]int{"a": nil}, got.Map)
	})

	t.Run("element is a pointer to a container", func(t *testing.T) {
		var empty []int
		got := struct {
			Map map[string]*[]int `default:"{\"x\": [9]}"`
		}{Map: map[string]*[]int{"a": &empty}}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Map["a"])
		assert.Nil(t, *got.Map["a"], "the caller's map is kept, and its element is left alone")
		assert.NotContains(t, got.Map, "x", "the tag is not applied to a map the caller provided")
	})
}
