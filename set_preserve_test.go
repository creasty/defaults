package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestSet_PreservesNonInitialScalar(t *testing.T) {
	type sample struct {
		String  string  `default:"default"`
		Int     int     `default:"1"`
		Float64 float64 `default:"1.5"`
	}

	got := sample{String: "caller", Int: 9, Float64: 9.5}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{String: "caller", Int: 9, Float64: 9.5}, got)
}

func TestSet_PreservesNonInitialScalarPointer(t *testing.T) {
	type sample struct {
		Int *int `default:"1"`
	}

	nine := 9
	got := sample{Int: &nine}
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Int)
	assert.Equal(t, 9, *got.Int)
}

func TestSet_PreservesNonInitialSlice(t *testing.T) {
	type sample struct {
		Ints []int `default:"[1, 2, 3]"`
	}

	got := sample{Ints: []int{9}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, []int{9}, got.Ints)
}

func TestSet_PreservesNonInitialMap(t *testing.T) {
	type sample struct {
		Map map[string]int `default:"{\"foo\": 123}"`
	}

	got := sample{Map: map[string]int{"bar": 9}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, map[string]int{"bar": 9}, got.Map)
}

func TestSet_PreservesNonInitialStruct(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Struct inner `default:"{\"Foo\": 1}"`
	}

	got := sample{Struct: inner{Foo: 9}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, inner{Name: "inner", Foo: 9}, got.Struct,
		"the tag does not overwrite a non-initial struct, but its fields still get their defaults")
}

func TestSet_PreservesNonInitialStructPointer(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Struct *inner `default:"{\"Foo\": 1}"`
	}

	got := sample{Struct: &inner{Foo: 9}}
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Struct)
	assert.Equal(t, inner{Name: "inner", Foo: 9}, *got.Struct)
}

// TestSet_CannotPreserveFalseBool pins the one type whose caller-provided value is indistinguishable
// from "unset": a bool. An explicit false is overwritten by `default:"true"`, and the way to keep it
// is a *bool, whose nil-ness carries that information.
//
// QUIRK: see https://github.com/creasty/defaults/pull/50 and
// https://github.com/creasty/defaults/pull/51.
func TestSet_CannotPreserveFalseBool(t *testing.T) {
	type sample struct {
		Bool    bool  `default:"true"`
		BoolPtr *bool `default:"true"`
	}

	explicitFalse := false
	got := sample{Bool: false, BoolPtr: &explicitFalse}
	require.NoError(t, defaults.Set(&got))

	assert.True(t, got.Bool, "a false bool is the zero value, so it is treated as unset")
	require.NotNil(t, got.BoolPtr)
	assert.False(t, *got.BoolPtr, "a non-nil *bool is left alone, false and all")
}

// TestSet_PreservesNonInitialZeroLengthContainers pins that "non-initial" means "not deep-equal to
// the zero value", so an allocated but empty slice or map is already non-initial and its tag is
// skipped.
func TestSet_PreservesNonInitialZeroLengthContainers(t *testing.T) {
	type sample struct {
		Slice []int          `default:"[1, 2, 3]"`
		Map   map[string]int `default:"{\"foo\": 123}"`
	}

	got := sample{Slice: []int{}, Map: map[string]int{}}
	require.NoError(t, defaults.Set(&got))

	assert.Empty(t, got.Slice)
	assert.Empty(t, got.Map)
}
