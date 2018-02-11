package defaults

import (
	"testing"
)

func TestCanUpdate(t *testing.T) {
	type st struct{ Int int }

	var myStructPtr *st

	pairs := map[interface{}]bool{
		0:            true,
		123:          false,
		float64(0):   true,
		float64(123): false,
		"":           true,
		"string":     false,
		false:        true,
		true:         false,
		st{}:         true,
		st{Int: 123}: false,
		myStructPtr:  true,
		&st{}:        false,
	}
	for input, expect := range pairs {
		output := CanUpdate(input)
		if output != expect {
			t.Errorf("CanUpdate(%v) returns %v, expected %v", input, output, expect)
		}
	}
}
