package defaults_test

import (
	"encoding/json"
	"testing"

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
			err := defaults.Set(tt.arg)

			require.Error(t, err)
			assert.EqualError(t, err, "not a struct pointer")
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
		assert.EqualError(t, defaults.Set(nil), "not a struct pointer")
	})

	t.Run("typed nil pointer", func(t *testing.T) {
		assert.EqualError(t, defaults.Set((*sample)(nil)), "not a struct pointer")
	})
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
		got := struct {
			Ptr *badSlice
		}{Ptr: &badSlice{}}

		require.Error(t, defaults.Set(&got))
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
