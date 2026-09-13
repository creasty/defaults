package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestMustSet_SetsDefaults(t *testing.T) {
	type sample struct {
		String string `default:"hello"`
	}

	var got sample
	require.NotPanics(t, func() {
		defaults.MustSet(&got)
	})

	assert.Equal(t, sample{String: "hello"}, got)
}

func TestMustSet_PanicsForNonPointer(t *testing.T) {
	type sample struct {
		String string `default:"hello"`
	}

	assert.PanicsWithValue(t, defaults.ErrInvalidType, func() {
		defaults.MustSet(sample{})
	})
}

func TestMustSet_PanicsForPointerToNonStruct(t *testing.T) {
	number := 1

	assert.PanicsWithValue(t, defaults.ErrInvalidType, func() {
		defaults.MustSet(&number)
	})
}

// TestMustSet_PanicsForNil pins that a nil panics with Set's error. It used to panic too, but
// from inside Set, before there was an error to panic with. See
// https://github.com/creasty/defaults/issues/69.
func TestMustSet_PanicsForNil(t *testing.T) {
	type sample struct {
		String string `default:"hello"`
	}

	t.Run("untyped nil", func(t *testing.T) {
		assert.PanicsWithValue(t, defaults.ErrInvalidType, func() {
			defaults.MustSet(nil)
		})
	})

	t.Run("typed nil pointer", func(t *testing.T) {
		assert.PanicsWithValue(t, defaults.ErrInvalidType, func() {
			defaults.MustSet((*sample)(nil))
		})
	})
}

func TestMustSet_PanicsForInvalidTag(t *testing.T) {
	got := struct {
		Ints []int `default:"[!]"`
	}{}

	assert.Panics(t, func() {
		defaults.MustSet(&got)
	})
}
