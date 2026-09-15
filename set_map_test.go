package defaults_test

import (
	"math"
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

// mapKeyHook runs hook when Set calls its setter, so a test can change the map holding it while Set
// walks that map. It is package-level only because it needs a method.
type mapKeyHook struct {
	Mark string
	hook func()
}

func (h *mapKeyHook) SetDefaults() {
	h.hook()
}

// TestSet_MapStructValueIsStoredBack pins that a struct held as a map value is stored back under its
// key once its copy is filled, whether or not anything changed. The structs here have nothing to
// fill, and their setter changes the map under their own key while the copy is filled: the copy
// stored afterwards brings a deleted key back and replaces a stored value.
//
// BUG: the store is a write to the map, even when nothing changed, so another goroutine reading the
// map while Set runs is a data race, which can end the program with "concurrent map read and map
// write". The maintainer keeps it as the contract, and the Set doc and the README say so: storing
// only a changed copy takes telling one from an unchanged one, which
// https://github.com/creasty/defaults/pull/101 did with a count of the writes made while the copy
// was filled and a byte comparison against a second copy.
func TestSet_MapStructValueIsStoredBack(t *testing.T) {
	t.Run("a deleted key comes back", func(t *testing.T) {
		got := struct {
			Map map[string]mapKeyHook
		}{Map: map[string]mapKeyHook{}}
		got.Map["a"] = mapKeyHook{hook: func() { delete(got.Map, "a") }}

		require.NoError(t, defaults.Set(&got))

		assert.Contains(t, got.Map, "a")
	})

	t.Run("a stored value is replaced", func(t *testing.T) {
		got := struct {
			Map map[string]mapKeyHook
		}{Map: map[string]mapKeyHook{}}
		got.Map["a"] = mapKeyHook{hook: func() { got.Map["a"] = mapKeyHook{Mark: "stored"} }}

		require.NoError(t, defaults.Set(&got))

		assert.Empty(t, got.Map["a"].Mark)
	})
}

// TestSet_MapContainerValueIsNotStoredBack covers a slice or map held as a map value. Set fills a copy
// of it too, but the copy refers to the same elements as the value in the map and those are filled in
// place, so the copy still equals what it was copied from, and storing it back could only undo a
// change a setter made under the value's key while the value's elements were filled. It is not
// stored, so such a setter keeps its change.
//
// The subtests where another goroutine reads the map rely on the race detector, which make test and
// make cover run: without -race they pass whether or not Set writes to the map.
func TestSet_MapContainerValueIsNotStoredBack(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}

	t.Run("slice values", func(t *testing.T) {
		t.Run("a deleted key stays deleted", func(t *testing.T) {
			got := struct {
				Map map[string][]mapKeyHook
			}{Map: map[string][]mapKeyHook{}}
			got.Map["a"] = []mapKeyHook{{hook: func() { delete(got.Map, "a") }}}

			require.NoError(t, defaults.Set(&got))

			assert.NotContains(t, got.Map, "a")
		})

		t.Run("a stored value stays", func(t *testing.T) {
			got := struct {
				Map map[string][]mapKeyHook
			}{Map: map[string][]mapKeyHook{}}
			got.Map["a"] = []mapKeyHook{{hook: func() { got.Map["a"] = []mapKeyHook{{Mark: "stored"}} }}}

			require.NoError(t, defaults.Set(&got))

			require.Len(t, got.Map["a"], 1)
			assert.Equal(t, "stored", got.Map["a"][0].Mark)
		})

		t.Run("another goroutine reads the map", func(t *testing.T) {
			got := struct {
				Map map[string][]inner
			}{Map: map[string][]inner{"a": {{}}}}

			done := make(chan struct{})
			go func() {
				defer close(done)
				_ = got.Map["a"]
			}()
			require.NoError(t, defaults.Set(&got))
			<-done

			assert.Equal(t, "inner", got.Map["a"][0].Name, "the element is filled in place")
		})
	})

	t.Run("map values", func(t *testing.T) {
		t.Run("a deleted key stays deleted", func(t *testing.T) {
			got := struct {
				Map map[string]map[string]mapKeyHook
			}{Map: map[string]map[string]mapKeyHook{}}
			got.Map["a"] = map[string]mapKeyHook{"b": {hook: func() { delete(got.Map, "a") }}}

			require.NoError(t, defaults.Set(&got))

			assert.NotContains(t, got.Map, "a")
		})

		t.Run("a stored value stays", func(t *testing.T) {
			got := struct {
				Map map[string]map[string]mapKeyHook
			}{Map: map[string]map[string]mapKeyHook{}}
			got.Map["a"] = map[string]mapKeyHook{"b": {hook: func() { got.Map["a"] = map[string]mapKeyHook{"stored": {}} }}}

			require.NoError(t, defaults.Set(&got))

			assert.Contains(t, got.Map["a"], "stored")
		})

		// The inner map holds pointers, so Set writes to neither map. With structs it would write to the
		// inner one, which the goroutine does not read.
		t.Run("another goroutine reads the map", func(t *testing.T) {
			got := struct {
				Map map[string]map[string]*inner
			}{Map: map[string]map[string]*inner{"a": {"b": {}}}}

			done := make(chan struct{})
			go func() {
				defer close(done)
				_ = got.Map["a"]
			}()
			require.NoError(t, defaults.Set(&got))
			<-done

			assert.Equal(t, "inner", got.Map["a"]["b"].Name, "the value is filled through")
		})
	})
}

// TestSet_MapEntryUnderNaNKeyIsSkipped pins that an entry whose key is NaN gets no defaults. Each
// value is looked up by its key, and NaN never equals itself, so the lookup finds nothing.
//
// QUIRK: Go's own m[k] cannot find such an entry either, and a value put back under NaN would be
// added beside the entry rather than replace it.
func TestSet_MapEntryUnderNaNKeyIsSkipped(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}

	got := struct {
		Map map[float64]*inner
	}{Map: map[float64]*inner{math.NaN(): {}, 1: {}}}

	require.NoError(t, defaults.Set(&got))

	require.Len(t, got.Map, 2)
	for key, value := range got.Map {
		if math.IsNaN(key) {
			assert.Empty(t, value.Name, "the entry under NaN is not reached")
		} else {
			assert.Equal(t, "inner", value.Name)
		}
	}
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
