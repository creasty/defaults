package defaults_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/creasty/defaults"
)

// TestCanUpdate covers what "can update" means: the value still holds the zero value of its own
// type. Note that an allocated-but-empty slice or map is therefore already non-initial.
func TestCanUpdate(t *testing.T) {
	type st struct {
		Int int
	}

	var nilPtr *st

	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"zero int", 0, true},
		{"non-zero int", 123, false},
		{"zero int8", int8(0), true},
		{"non-zero uint", uint(1), false},
		{"zero float", float64(0), true},
		{"non-zero float", float64(123), false},
		{"empty string", "", true},
		{"non-empty string", "string", false},
		{"false", false, true},
		{"true", true, false},
		{"zero duration", time.Duration(0), true},
		{"non-zero duration", time.Second, false},
		{"zero struct", st{}, true},
		{"non-zero struct", st{Int: 123}, false},
		{"nil pointer", nilPtr, true},
		{"pointer to zero struct", &st{}, false},
		{"nil slice", []int(nil), true},
		{"empty slice", []int{}, false},
		{"filled slice", []int{1}, false},
		{"nil map", map[string]int(nil), true},
		{"empty map", map[string]int{}, false},
		{"filled map", map[string]int{"a": 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, defaults.CanUpdate(tt.value))
		})
	}
}

// TestCanUpdate_FloatEdges pins the two floats where a per-kind zero check and a comparison against
// the zero value could plausibly disagree: a negative zero counts as zero, a NaN does not. Both hold
// inside a struct too, which is where reflect has historically drawn the line differently
// (https://github.com/golang/go/issues/61827).
func TestCanUpdate_FloatEdges(t *testing.T) {
	type st struct {
		Float float64
	}

	negZero := math.Copysign(0, -1)
	nan := math.NaN()

	assert.True(t, defaults.CanUpdate(negZero), "a negative zero is a zero")
	assert.True(t, defaults.CanUpdate(st{Float: negZero}), "and so is a struct holding one")
	assert.False(t, defaults.CanUpdate(nan), "a NaN is not equal to zero")
	assert.False(t, defaults.CanUpdate(st{Float: nan}), "nor is a struct holding one")
}

// TestCanUpdate_Nil covers a nil, which carries no type to compare against: there is nothing there to
// preserve, so it is updatable. It used to panic inside reflect instead.
//
// A nil value of an interface type collapses to the same invalid reflect.Value as an untyped nil, and
// that is how this was reached in practice — a nil interface field read by a SetDefaults
// implementation. See https://github.com/creasty/defaults/issues/47.
func TestCanUpdate_Nil(t *testing.T) {
	var nilError error

	assert.True(t, defaults.CanUpdate(nil), "an untyped nil")
	assert.True(t, defaults.CanUpdate(nilError), "a nil value of an interface type")
}
