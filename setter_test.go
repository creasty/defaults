package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// setterBar writes unconditionally, so a test can watch a setter clobber a caller's value. Every
// type in this file is package-level because Go does not allow methods on function-local types.
type setterBar struct {
	Foo  int
	Bar  int
	Name string `default:"name"`
}

func (s *setterBar) SetDefaults() {
	s.Bar = 456
}

// setterGuarded follows the README's advice of checking CanUpdate before writing.
type setterGuarded struct {
	Value int
}

func (s *setterGuarded) SetDefaults() {
	if defaults.CanUpdate(s.Value) {
		s.Value = 456
	}
}

// setterCounter counts its own invocations.
type setterCounter struct {
	Calls int
}

func (s *setterCounter) SetDefaults() {
	s.Calls++
}

// SetterInner is embedded by pointer below, which is why its name is exported: an embedded field
// takes the name of its type, and an unexported one would be beyond reflect's reach.
type SetterInner struct {
	InnerInt int `default:"-"`
}

func (s *SetterInner) SetDefaults() {
	if defaults.CanUpdate(s.InnerInt) {
		s.InnerInt = 1
	}
}

// setterOuter is the README's shape: a field opted out of tags and filled by the struct's own
// setter, next to an embedded pointer that has a setter of its own.
type setterOuter struct {
	OuterInt     int `default:"-"`
	*SetterInner `default:"{}"`
}

func (s *setterOuter) SetDefaults() {
	if defaults.CanUpdate(s.OuterInt) {
		s.OuterInt = 1
	}
}

// setterLevel is a non-struct type with a setter. It is the one shape whose setter is invoked only
// by the pointer branch of setField, because the struct recursion never sees it — so a change there
// that looks redundant would silently stop calling it. See
// https://github.com/creasty/defaults/issues/67.
type setterLevel int

func (l *setterLevel) SetDefaults() {
	*l += 100
}

func TestSetter_CalledOnRoot(t *testing.T) {
	var got setterBar

	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 456, got.Bar)
	assert.Equal(t, "name", got.Name, "tags are applied before the setter runs")
}

func TestSetter_CalledOnStructField(t *testing.T) {
	type sample struct {
		Struct setterBar
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 456, got.Struct.Bar)
}

func TestSetter_CalledOnPointerField(t *testing.T) {
	type sample struct {
		Ptr *setterBar `default:"{}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Ptr)
	assert.Equal(t, 456, got.Ptr.Bar)
}

func TestSetter_CalledOnSliceElements(t *testing.T) {
	type sample struct {
		FromTag  []setterBar `default:"[{}]"`
		FromCall []setterBar
	}

	got := sample{FromCall: []setterBar{{Foo: 1}}}
	require.NoError(t, defaults.Set(&got))

	require.Len(t, got.FromTag, 1)
	require.Len(t, got.FromCall, 1)
	assert.Equal(t, 456, got.FromTag[0].Bar)
	assert.Equal(t, 456, got.FromCall[0].Bar)
}

func TestSetter_CalledOnMapElements(t *testing.T) {
	type sample struct {
		Map map[string]setterBar
	}

	got := sample{Map: map[string]setterBar{"a": {Foo: 1}}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 456, got.Map["a"].Bar, "the defaulted copy is written back into the map")
}

func TestSetter_CalledOnMapPointerElements(t *testing.T) {
	type sample struct {
		Map map[string]*setterBar
	}

	got := sample{Map: map[string]*setterBar{"a": {Foo: 1}}}
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Map["a"])
	assert.Equal(t, 456, got.Map["a"].Bar)
}

// TestSetter_OverwritesCallerValueWhenUnguarded pins that a setter is trusted absolutely: unlike a
// tag, it is not checked against the field's current value.
func TestSetter_OverwritesCallerValueWhenUnguarded(t *testing.T) {
	type sample struct {
		Struct setterBar
	}

	got := sample{Struct: setterBar{Bar: 5}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 456, got.Struct.Bar, "an unguarded setter clobbers what the caller passed")
}

func TestSetter_CanUpdateGuardPreservesCallerValue(t *testing.T) {
	type sample struct {
		Struct setterGuarded
	}

	got := sample{Struct: setterGuarded{Value: 9}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 9, got.Struct.Value, "CanUpdate is what makes a setter respect the caller")
}

func TestSetter_EmbeddedPointerStruct(t *testing.T) {
	var got setterOuter

	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 1, got.OuterInt)
	require.NotNil(t, got.SetterInner)
	assert.Equal(t, 1, got.InnerInt, "the embedded pointer is allocated and its setter runs too")
}

// TestSetter_InvocationCount pins how many times Set calls a setter for each shape. A pointer field
// gets two calls — once from the Set recursion, once from the pointer branch that wraps it — which
// is why a SetDefaults implementation has to be idempotent, and why the README recommends guarding
// it with CanUpdate.
//
// QUIRK(defaults.go:164): the double call is not by design, just how the recursion falls out.
func TestSetter_InvocationCount(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		var got setterCounter

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Calls)
	})

	t.Run("struct field", func(t *testing.T) {
		var got struct {
			Struct setterCounter
		}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Struct.Calls)
	})

	t.Run("pointer field", func(t *testing.T) {
		var got struct {
			Ptr *setterCounter `default:"{}"`
		}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Ptr)
		assert.Equal(t, 2, got.Ptr.Calls)
	})

	t.Run("slice element", func(t *testing.T) {
		got := struct {
			Slice []setterCounter
		}{Slice: []setterCounter{{}}}

		require.NoError(t, defaults.Set(&got))

		require.Len(t, got.Slice, 1)
		assert.Equal(t, 1, got.Slice[0].Calls)
	})

	t.Run("map element", func(t *testing.T) {
		got := struct {
			Map map[string]setterCounter
		}{Map: map[string]setterCounter{"a": {}}}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Map["a"].Calls)
	})

	t.Run("map element behind a pointer", func(t *testing.T) {
		got := struct {
			Map map[string]*setterCounter
		}{Map: map[string]*setterCounter{"a": {}}}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Map["a"])
		assert.Equal(t, 1, got.Map["a"].Calls)
	})
}

func TestSetter_CalledOnPointerToNonStruct(t *testing.T) {
	t.Run("behind a pointer", func(t *testing.T) {
		got := struct {
			Level *setterLevel `default:"3"`
		}{}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Level)
		assert.Equal(t, setterLevel(103), *got.Level, "the tag sets 3, then the setter adds 100")
	})

	t.Run("not behind a pointer", func(t *testing.T) {
		got := struct {
			Level setterLevel `default:"3"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, setterLevel(3), got.Level, "a non-struct value field never reaches a setter")
	})
}

// TestSetter_RunsAgainOnASecondSet pins that Set is not idempotent as far as setters go: calling it
// twice calls them twice over, which is the other half of why a setter has to be idempotent itself.
func TestSetter_RunsAgainOnASecondSet(t *testing.T) {
	var got struct {
		Value setterCounter
		Ptr   *setterCounter `default:"{}"`
	}

	require.NoError(t, defaults.Set(&got))
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Ptr)
	assert.Equal(t, 2, got.Value.Calls)
	assert.Equal(t, 4, got.Ptr.Calls)
}
