package defaults_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// mapGuarded's setter writes only while its field is zero, as the Setter example does. Its type,
// mapReplacer's and mapHook's are package-level only because they need a method.
type mapGuarded struct {
	Name string
}

func (g *mapGuarded) SetDefaults() {
	if defaults.CanUpdate(g.Name) {
		g.Name = "guarded"
	}
}

// mapReplacer's setter replaces its pointer with a new one to an equal value: a change that only
// the pointer's identity shows.
type mapReplacer struct {
	Ptr *int
}

func (r *mapReplacer) SetDefaults() {
	n := *r.Ptr
	r.Ptr = &n
}

// mapHook runs a function when Set calls its setter, so a test can act while a map value is filled.
type mapHook struct {
	run func()
}

func (h *mapHook) SetDefaults() {
	h.run()
}

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

// TestSet_FilledMapIsNotWrittenTo pins that Set writes nothing into a map it has nothing to fill.
// Every struct, slice and map value used to be put back after the recursion whether it changed or
// not, so Set on a struct holding a map someone else was reading could crash the program with a
// concurrent map read and map write.
//
// The assertion is the race detector's, which `make test` runs: a write to the map races with the
// read in the goroutine, and nothing else does.
func TestSet_FilledMapIsNotWrittenTo(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}

	t.Run("struct values", func(t *testing.T) {
		got := struct {
			Map map[string]inner
		}{Map: map[string]inner{"a": {Name: "kept"}}}

		read := make(chan struct{})
		go func() {
			defer close(read)
			_ = got.Map["a"]
		}()

		require.NoError(t, defaults.Set(&got))
		<-read
	})

	t.Run("struct values whose setter changes nothing", func(t *testing.T) {
		got := struct {
			Map map[string]mapGuarded
		}{Map: map[string]mapGuarded{"a": {Name: "kept"}}}

		read := make(chan struct{})
		go func() {
			defer close(read)
			_ = got.Map["a"]
		}()

		require.NoError(t, defaults.Set(&got))
		<-read
	})

	t.Run("slice values", func(t *testing.T) {
		got := struct {
			Map map[string][]inner
		}{Map: map[string][]inner{"a": {{Name: "kept"}}}}

		read := make(chan struct{})
		go func() {
			defer close(read)
			_ = got.Map["a"]
		}()

		require.NoError(t, defaults.Set(&got))
		<-read
	})

	t.Run("map values", func(t *testing.T) {
		got := struct {
			Map map[string]map[string]inner
		}{Map: map[string]map[string]inner{"a": {"b": {Name: "kept"}}}}

		read := make(chan struct{})
		go func() {
			defer close(read)
			_ = got.Map["a"]
		}()

		require.NoError(t, defaults.Set(&got))
		<-read
	})
}

// TestSet_ChangedMapStructValueIsWrittenBack pins that a struct value in a map is put back whenever
// its copy changed at all, even where reflect.DeepEqual would see no change: a zero whose sign
// changed, and a pointer replaced by a new one to an equal value. Deciding with DeepEqual would
// drop both. A copy that did not change is not put back, so it does not undo a store made under its
// key while it was filled.
func TestSet_ChangedMapStructValueIsWrittenBack(t *testing.T) {
	t.Run("a negative zero", func(t *testing.T) {
		type inner struct {
			F float64 `default:"-0"`
		}

		got := struct {
			Map map[string]inner
		}{Map: map[string]inner{"a": {}}}

		require.NoError(t, defaults.Set(&got))

		assert.True(t, math.Signbit(got.Map["a"].F))
	})

	t.Run("a pointer replaced by an equal one", func(t *testing.T) {
		n := 1
		got := struct {
			Map map[string]mapReplacer
		}{Map: map[string]mapReplacer{"a": {Ptr: &n}}}

		require.NoError(t, defaults.Set(&got))

		assert.NotSame(t, &n, got.Map["a"].Ptr)
		assert.Equal(t, 1, *got.Map["a"].Ptr)
	})

	t.Run("a store made while the copy was filled is kept", func(t *testing.T) {
		type entry struct {
			Hook mapHook
			N    int
		}

		got := struct {
			Map map[string]entry
		}{Map: map[string]entry{}}
		got.Map["a"] = entry{Hook: mapHook{run: func() {
			got.Map["a"] = entry{N: 5}
		}}}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 5, got.Map["a"].N, "the unchanged copy does not overwrite the store")
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
