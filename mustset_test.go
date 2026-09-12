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

	assert.PanicsWithError(t, "not a struct pointer", func() {
		defaults.MustSet(sample{})
	})
}

func TestMustSet_PanicsForPointerToNonStruct(t *testing.T) {
	number := 1

	assert.PanicsWithError(t, "not a struct pointer", func() {
		defaults.MustSet(&number)
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
