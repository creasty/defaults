package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// optOutSetter records whether its setter ran, so a test can tell whether `default:"-"` suppresses
// the setter along with the tags. It is package-level only because it needs a method.
type optOutSetter struct {
	Called bool
	Name   string `default:"name"`
}

func (s *optOutSetter) SetDefaults() {
	s.Called = true
}

// optOutRoot is the documented way to give an opted-out field a dynamic default: the tag says
// "hands off", and the struct's own setter fills it in.
type optOutRoot struct {
	Int int `default:"-"`
}

func (s *optOutRoot) SetDefaults() {
	if defaults.CanUpdate(s.Int) {
		s.Int = 1
	}
}

func TestSet_OptOutSkipsScalar(t *testing.T) {
	type sample struct {
		Int    int    `default:"-"`
		String string `default:"-"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{}, got)
}

func TestSet_OptOutSkipsContainers(t *testing.T) {
	type sample struct {
		Slice []string       `default:"-"`
		Map   map[string]int `default:"-"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Nil(t, got.Slice)
	assert.Nil(t, got.Map)
}

func TestSet_OptOutSkipsPointer(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		Int    *int   `default:"-"`
		Struct *inner `default:"-"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Nil(t, got.Int)
	assert.Nil(t, got.Struct)
}

// TestSet_OptOutSkipsStructRecursionAndSetter covers the reach of `-` on a struct field: it is not
// descended into, so neither the nested tags nor the nested setter run.
func TestSet_OptOutSkipsStructRecursionAndSetter(t *testing.T) {
	type sample struct {
		Struct optOutSetter `default:"-"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Empty(t, got.Struct.Name, "nested tags are not applied")
	assert.False(t, got.Struct.Called, "the nested setter is not called either")
}

// TestSet_OptOutDoesNotAffectOwnSetter covers the other side: opting a field out does not opt its
// own struct out of having its setter called.
func TestSet_OptOutDoesNotAffectOwnSetter(t *testing.T) {
	var got optOutRoot

	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 1, got.Int, "the setter fills what the tag opted out of")
}

func TestSet_OptOutPreservesCallerValue(t *testing.T) {
	got := optOutRoot{Int: 9}

	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 9, got.Int, "CanUpdate keeps the setter from clobbering it")
}
