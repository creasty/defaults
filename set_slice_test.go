package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestSet_EmptySliceTagCreatesEmptySlice(t *testing.T) {
	type sample struct {
		Slice []string `default:"[]"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.NotNil(t, got.Slice, "an allocated slice, not a nil one")
	assert.Empty(t, got.Slice)
}

func TestSet_SliceTagJSON(t *testing.T) {
	type sample struct {
		Strings []string `default:"[\"foo\", \"bar\"]"`
		Ints    []int    `default:"[1, 2, 3]"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, []string{"foo", "bar"}, got.Strings)
	assert.Equal(t, []int{1, 2, 3}, got.Ints)
}

func TestSet_SliceOfPointersFromTag(t *testing.T) {
	type sample struct {
		Ptrs []*int `default:"[1, 2, 3]"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.Len(t, got.Ptrs, 3)
	for i, want := range []int{1, 2, 3} {
		require.NotNil(t, got.Ptrs[i])
		assert.Equal(t, want, *got.Ptrs[i])
	}
}

func TestSet_NamedSliceType(t *testing.T) {
	type mySlice []int
	type sample struct {
		Empty mySlice `default:"[]"`
		JSON  mySlice `default:"[1, 2]"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.NotNil(t, got.Empty)
	assert.Empty(t, got.Empty)
	assert.Equal(t, mySlice{1, 2}, got.JSON)
}

func TestSet_UntaggedSliceStaysNil(t *testing.T) {
	type sample struct {
		Slice []string
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Nil(t, got.Slice)
}

func TestSet_SliceOfStructsFromTag(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Empty  []inner `default:"[{}]"`
		Filled []inner `default:"[{\"Foo\": 123}]"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, []inner{{Name: "inner"}}, got.Empty, "an element created by the tag still gets its own defaults")
	assert.Equal(t, []inner{{Name: "inner", Foo: 123}}, got.Filled)
}

// TestSet_SliceTagOverridesElementDefault covers precedence between a parent's tag and a child's
// own default: the parent wins.
func TestSet_SliceTagOverridesElementDefault(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		Slice []inner `default:"[{\"Name\": \"changed\"}]"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, []inner{{Name: "changed"}}, got.Slice)
}

// TestSet_SliceElementsGetDefaults covers a slice the caller filled in: Set descends into each
// element even though the field carries no tag.
func TestSet_SliceElementsGetDefaults(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Slice []inner
	}

	got := sample{Slice: []inner{{Foo: 1}, {Name: "kept", Foo: 2}}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, []inner{
		{Name: "inner", Foo: 1},
		{Name: "kept", Foo: 2},
	}, got.Slice)
}

// TestSet_SliceOfScalarsIsLeftAlone pins that elements of a scalar slice are never rewritten: a
// zero element is not a field with a tag, so nothing applies to it.
func TestSet_SliceOfScalarsIsLeftAlone(t *testing.T) {
	type sample struct {
		Ints []int `default:"[0, 2]"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, []int{0, 2}, got.Ints)
}
