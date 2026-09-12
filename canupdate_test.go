package defaults_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/creasty/defaults"
)

// TestCanUpdate covers what "can update" means: deep-equal to the zero value of its own type. Note
// that an allocated-but-empty slice or map is therefore already non-initial.
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

// TestCanUpdate_PanicsOnUntypedNil pins that an untyped nil has no type to compare against, so the
// reflect call behind CanUpdate fails.
//
// QUIRK: a nil could reasonably be reported as updatable. See
// https://github.com/creasty/defaults/pull/64.
func TestCanUpdate_PanicsOnUntypedNil(t *testing.T) {
	assert.Panics(t, func() {
		defaults.CanUpdate(nil)
	})
}
