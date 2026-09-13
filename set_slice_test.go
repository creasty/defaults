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

// TestSet_ArraysAreLeftAlone pins that reflect.Array has no case in setField, which makes an array
// behave unlike the slice of the same element type: its tag is dropped without an error, and its
// elements are not descended into. Only a tag that the array type's own unmarshaler rejects is an
// error; that is TestSet_FailingUnmarshalerWithNothingToFallBackTo.
func TestSet_ArraysAreLeftAlone(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		Ints     [2]int   `default:"[1, 2]"`
		Structs  [1]inner `default:"[{}]"`
		Provided [1]inner
	}

	got := sample{Provided: [1]inner{{}}}
	require.NoError(t, defaults.Set(&got), "an array tag is ignored rather than rejected")

	assert.Equal(t, [2]int{}, got.Ints, "the tag is dropped, where a []int would have been filled")
	assert.Equal(t, [1]inner{}, got.Structs)
	assert.Equal(t, [1]inner{}, got.Provided, "array elements are not descended into, where slice elements are")
}

// TestSet_ByteSlice covers []byte, where encoding/json accepts two spellings and rejects bare text.
func TestSet_ByteSlice(t *testing.T) {
	t.Run("json array", func(t *testing.T) {
		got := struct {
			B []byte `default:"[104, 105]"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, []byte("hi"), got.B)
	})

	t.Run("base64 string", func(t *testing.T) {
		got := struct {
			B []byte `default:"\"aGk=\""`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, []byte("hi"), got.B)
	})

	t.Run("bare text is an error", func(t *testing.T) {
		got := struct {
			B []byte `default:"hi"`
		}{}

		require.Error(t, defaults.Set(&got), "a []byte tag goes through encoding/json, so it is not a plain string")
	})
}
