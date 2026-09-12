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
