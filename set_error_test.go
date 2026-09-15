package defaults_test

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestSet_RejectsNonStructPointer(t *testing.T) {
	type sample struct {
		Int int `default:"1"`
	}

	number := 1
	slice := []int{1}
	mapping := map[string]int{"a": 1}

	tests := []struct {
		name string
		arg  interface{}
	}{
		{"a value, not a pointer", 1},
		{"a struct value", sample{}},
		{"a string", "hello"},
		{"a map", mapping},
		{"a pointer to an int", &number},
		{"a pointer to a slice", &slice},
		{"a pointer to a map", &mapping},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, defaults.Set(tt.arg), defaults.ErrInvalidType)
		})
	}
}

// TestSet_RejectsNil covers a nil, typed or not, which has no struct behind it to fill: it is
// rejected with the same error as any other non-struct-pointer. It used to panic instead.
//
// The typed nil is the case real code is likely to hit — a config pointer that was never
// allocated. See https://github.com/creasty/defaults/issues/69.
func TestSet_RejectsNil(t *testing.T) {
	type sample struct {
		Int int `default:"1"`
	}

	t.Run("untyped nil", func(t *testing.T) {
		assert.ErrorIs(t, defaults.Set(nil), defaults.ErrInvalidType)
	})

	t.Run("typed nil pointer", func(t *testing.T) {
		assert.ErrorIs(t, defaults.Set((*sample)(nil)), defaults.ErrInvalidType)
	})
}

// TestSet_InvalidTypeKeepsItsMessage pins the message of ErrInvalidType. Until the error was
// exported, matching on its message was the only way to tell it apart, so code written then may
// still be doing it. See https://github.com/creasty/defaults/issues/70.
func TestSet_InvalidTypeKeepsItsMessage(t *testing.T) {
	assert.EqualError(t, defaults.Set(struct{}{}), "not a struct pointer")
}

func TestSet_InvalidJSONInTag(t *testing.T) {
	type badSlice struct {
		Ints []int `default:"[!]"`
	}

	tests := []struct {
		name string
		ptr  interface{}
	}{
		{
			name: "slice tag",
			ptr: &struct {
				Ints []int `default:"[!]"`
			}{},
		},
		{
			name: "map tag",
			ptr: &struct {
				Map map[string]int `default:"{1}"`
			}{},
		},
		{
			name: "struct tag",
			ptr: &struct {
				Struct struct{ Ints []int } `default:"{!}"`
			}{},
		},
		{
			name: "nested field tag",
			ptr: &struct {
				Struct struct {
					Ints []int `default:"[!]"`
				}
			}{},
		},
		{
			name: "slice element",
			ptr:  &struct{ Slice []badSlice }{Slice: []badSlice{{}}},
		},
		{
			name: "map element",
			ptr:  &struct{ Map map[string]badSlice }{Map: map[string]badSlice{"a": {}}},
		},
		{
			name: "map element behind a pointer",
			ptr:  &struct{ Map map[string]*badSlice }{Map: map[string]*badSlice{"a": {}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := defaults.Set(tt.ptr)

			require.Error(t, err)
			var syntaxErr *json.SyntaxError
			assert.ErrorAs(t, err, &syntaxErr, "the decoder's error should reach the caller unwrapped")
		})
	}
}

// TestSet_PointerFieldErrorIsReported covers a *struct field whose nested tag is invalid. That error
// used to be discarded, alone among the recursions, so a malformed tag went unreported depending on
// whether the field it sat behind was a pointer. See https://github.com/creasty/defaults/issues/68.
func TestSet_PointerFieldErrorIsReported(t *testing.T) {
	type badSlice struct {
		Ints []int `default:"[!]"`
	}

	t.Run("pointer created from the tag", func(t *testing.T) {
		got := struct {
			Ptr *badSlice `default:"{}"`
		}{}

		require.Error(t, defaults.Set(&got))
	})

	t.Run("pointer provided by the caller", func(t *testing.T) {
		ptr := &badSlice{}
		got := struct {
			Ptr *badSlice
		}{Ptr: ptr}

		require.Error(t, defaults.Set(&got))
		assert.Same(t, ptr, got.Ptr, "only a pointer the tag allocated is put back")
	})
}

// TestSet_ErrorStopsAtTheFirstField pins that Set gives up on the first failing field rather than
// collecting errors, so later fields keep their zero values.
func TestSet_ErrorStopsAtTheFirstField(t *testing.T) {
	got := struct {
		Bad   []int  `default:"[!]"`
		After string `default:"hello"`
	}{}

	require.Error(t, defaults.Set(&got))
	assert.Empty(t, got.After)
}

// TestSet_FailingDefaultLeavesZeroValuesZero pins that a value Set found zero is zero again when a
// default below it fails, so a second Set meets the same tag and fails again. Set used to leave what
// it had filled before the failure: a pointer or container the tag allocated, a struct encoding/json
// decoded partway, the fields of a struct filled before one of them failed, or what an unmarshaler
// wrote before rejecting the tag. The value was no longer zero, so a second Set could skip its tag
// and return nil, or, below a tag that recurses, grow the value by another level.
func TestSet_FailingDefaultLeavesZeroValuesZero(t *testing.T) {
	type point struct {
		X int
		Y int
	}
	type bad struct {
		Ints []int `default:"[!]"`
	}
	type partly struct {
		Before int `default:"1"`
		After  bad
	}

	tests := []struct {
		name string
		ptr  interface{}
	}{
		{"a pointer to a scalar", &struct {
			V *time.Duration `default:"10 s"`
		}{}},
		{"a pointer to a slice", &struct {
			V *[]int `default:"[!]"`
		}{}},
		{"a pointer to a struct", &struct {
			V *point `default:"{!}"`
		}{}},
		{"a struct decoded partway", &struct {
			V point `default:"{\"X\": 1, \"Y\": \"two\"}"`
		}{}},
		{"a struct filled partway", &struct {
			V partly
		}{}},
		{"a slice the tag allocated", &struct {
			V []bad `default:"[{}]"`
		}{}},
		{"a map the tag allocated", &struct {
			V map[string]bad `default:"{\"a\": {}}"`
		}{}},
		{"an unmarshaler that wrote part of a value", &struct {
			V big.Int `default:"12x"`
		}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, defaults.Set(tt.ptr))
			assert.Zero(t, reflect.ValueOf(tt.ptr).Elem().Interface(), "what Set found zero is zero again")
			require.Error(t, defaults.Set(tt.ptr), "so a second Set fails again")
		})
	}
}

// TestSet_FailingDefaultKeepsWhatWasNotZero pins the other side: a value Set did not find zero is not
// put back, so the caller's slice or map survives a default that fails below it, and so does a field
// Set filled before reaching the one that failed. TestSet_PointerFieldErrorIsReported pins it for a
// pointer.
func TestSet_FailingDefaultKeepsWhatWasNotZero(t *testing.T) {
	type bad struct {
		Ints []int `default:"[!]"`
	}

	t.Run("the caller's slice", func(t *testing.T) {
		got := struct {
			Items []bad
		}{Items: []bad{{}}}

		require.Error(t, defaults.Set(&got))
		assert.Len(t, got.Items, 1)
	})

	t.Run("the caller's map", func(t *testing.T) {
		got := struct {
			Items map[string]bad
		}{Items: map[string]bad{"a": {}}}

		require.Error(t, defaults.Set(&got))
		assert.Len(t, got.Items, 1)
	})

	t.Run("a field filled before the one that failed", func(t *testing.T) {
		got := struct {
			Before int `default:"1"`
			After  bad
		}{}

		require.Error(t, defaults.Set(&got))
		assert.Equal(t, 1, got.Before)
	})
}

// TestSet_WellFormedJSONOfTheWrongShape covers the other way decoding fails: valid JSON that cannot
// land in the field's type. It surfaces as *json.UnmarshalTypeError, not a syntax error.
func TestSet_WellFormedJSONOfTheWrongShape(t *testing.T) {
	got := struct {
		Ints []int `default:"{}"`
	}{}

	err := defaults.Set(&got)

	require.Error(t, err)
	var typeErr *json.UnmarshalTypeError
	assert.ErrorAs(t, err, &typeErr)
}

// TestSet_WhitespaceTagIsNotEmpty pins that a stray space is a value rather than an absence: a
// string takes it verbatim, and every container tries to decode it as JSON and fails.
func TestSet_WhitespaceTagIsNotEmpty(t *testing.T) {
	t.Run("a string takes it", func(t *testing.T) {
		got := struct {
			S string `default:" "`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, " ", got.S)
	})

	tests := []struct {
		name string
		ptr  interface{}
	}{
		{"slice", &struct {
			V []int `default:" "`
		}{}},
		{"map", &struct {
			V map[string]int `default:" "`
		}{}},
		{"struct", &struct {
			V struct{ I int } `default:" "`
		}{}},
	}

	for _, tt := range tests {
		t.Run("a "+tt.name+" fails to decode it", func(t *testing.T) {
			assert.Error(t, defaults.Set(tt.ptr))
		})
	}
}
