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

// SetterCounter counts its own invocations. Its name is exported because it is embedded below, for
// the same reason as SetterInner's.
type SetterCounter struct {
	Calls int
}

func (s *SetterCounter) SetDefaults() {
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

// SetterLevel is a non-struct type with a setter. The struct recursion never reaches it, so behind a
// pointer its only call is the one the pointer branch of setField makes itself — the call that is
// redundant for a struct, and skipped for one, but cannot be dropped outright. See
// https://github.com/creasty/defaults/issues/67. Its name is exported because it is embedded below.
type SetterLevel int

func (l *SetterLevel) SetDefaults() {
	*l += 100
}

// setterByValue declares its setter on a value receiver. The setter works on a copy, so it reports
// its call through the pointer the copy shares.
type setterByValue struct {
	Calls *int
}

func (s setterByValue) SetDefaults() {
	*s.Calls++
}

// setterGeneric is a generic type with a setter of its own.
type setterGeneric[T any] struct {
	Value T `default:"value"`
	Calls int
}

func (s *setterGeneric[T]) SetDefaults() {
	s.Calls++
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

// TestSetter_EmbeddedPointerStruct pins that a struct's own setter and an embedded one both run.
// setterOuter declares a SetDefaults, which hides SetterInner's from setterOuter's method set, so the
// two are separate setters. A struct that has SetDefaults only by promotion gets no call of its own;
// see TestSetter_InvocationCount.
func TestSetter_EmbeddedPointerStruct(t *testing.T) {
	var got setterOuter

	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 1, got.OuterInt)
	require.NotNil(t, got.SetterInner)
	assert.Equal(t, 1, got.InnerInt, "the embedded pointer is allocated and its setter runs too")
}

// TestSetter_NotCalledThroughNilEmbeddedPointer pins that an untagged nil embedded pointer gets no
// setter, the same as a named one. Go promotes the setter to the struct all the same, and Set used to
// call it through the struct, on the nil pointer — a panic for any setter that touches its fields.
func TestSetter_NotCalledThroughNilEmbeddedPointer(t *testing.T) {
	var got struct {
		*SetterCounter
	}

	var err error
	require.NotPanics(t, func() {
		err = defaults.Set(&got)
	})

	require.NoError(t, err)
	assert.Nil(t, got.SetterCounter)
}

// TestSetter_NotCalledThroughEmbeddedInterface pins that an embedded interface gets no setter, the
// same as a named one. Go promotes the interface's SetDefaults to the struct, and Set used to call it
// through the struct: on whatever the interface held, or on nil, which panicked.
func TestSetter_NotCalledThroughEmbeddedInterface(t *testing.T) {
	t.Run("holding a setter", func(t *testing.T) {
		held := &SetterCounter{}
		got := struct {
			defaults.Setter
		}{Setter: held}

		require.NoError(t, defaults.Set(&got))

		assert.Zero(t, held.Calls)
	})

	t.Run("nil", func(t *testing.T) {
		var got struct {
			defaults.Setter
		}

		var err error
		require.NotPanics(t, func() {
			err = defaults.Set(&got)
		})

		require.NoError(t, err)
	})
}

// TestSetter_InvocationCount pins how many times Set calls a setter for each shape below: once.
//
// A struct behind a pointer used to get a second call, from the pointer branch of setField after the
// recursion into the struct had already made one, so a setter that was not idempotent applied
// itself twice. See https://github.com/creasty/defaults/issues/67.
//
// An embedded struct got a second call by another route. Go promotes the embedded type's SetDefaults
// to the struct that embeds it, so Set's call on that struct, made after the recursion into the field
// had already called it, ran the same setter on the same field again — once more for every level of
// embedding.
func TestSetter_InvocationCount(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		var got SetterCounter

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Calls)
	})

	t.Run("struct field", func(t *testing.T) {
		var got struct {
			Struct SetterCounter
		}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Struct.Calls)
	})

	t.Run("pointer field", func(t *testing.T) {
		var got struct {
			Ptr *SetterCounter `default:"{}"`
		}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Ptr)
		assert.Equal(t, 1, got.Ptr.Calls)
	})

	t.Run("pointer field the caller allocated", func(t *testing.T) {
		got := struct {
			Ptr *SetterCounter
		}{Ptr: &SetterCounter{}}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Ptr.Calls)
	})

	t.Run("slice element", func(t *testing.T) {
		got := struct {
			Slice []SetterCounter
		}{Slice: []SetterCounter{{}}}

		require.NoError(t, defaults.Set(&got))

		require.Len(t, got.Slice, 1)
		assert.Equal(t, 1, got.Slice[0].Calls)
	})

	t.Run("slice element behind a pointer", func(t *testing.T) {
		got := struct {
			Slice []*SetterCounter
		}{Slice: []*SetterCounter{{}}}

		require.NoError(t, defaults.Set(&got))

		require.Len(t, got.Slice, 1)
		assert.Equal(t, 1, got.Slice[0].Calls)
	})

	t.Run("map element", func(t *testing.T) {
		got := struct {
			Map map[string]SetterCounter
		}{Map: map[string]SetterCounter{"a": {}}}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Map["a"].Calls)
	})

	t.Run("map element behind a pointer", func(t *testing.T) {
		got := struct {
			Map map[string]*SetterCounter
		}{Map: map[string]*SetterCounter{"a": {}}}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Map["a"])
		assert.Equal(t, 1, got.Map["a"].Calls)
	})

	t.Run("embedded struct", func(t *testing.T) {
		var got struct {
			SetterCounter
		}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Calls)
	})

	t.Run("embedded pointer", func(t *testing.T) {
		var got struct {
			*SetterCounter `default:"{}"`
		}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.SetterCounter)
		assert.Equal(t, 1, got.Calls)
	})

	t.Run("embedded two levels deep", func(t *testing.T) {
		type Middle struct {
			SetterCounter
		}

		var got struct {
			Middle
		}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, 1, got.Calls)
	})
}

// TestSetter_CalledOnPointerToNonStruct pins the setter calls a non-struct type gets: one behind a
// pointer, and none otherwise. Embedding the field changes neither; it used to add one, because Go
// promotes the embedded type's SetDefaults to the struct and Set called it through the struct too.
func TestSetter_CalledOnPointerToNonStruct(t *testing.T) {
	t.Run("behind a pointer", func(t *testing.T) {
		got := struct {
			Level *SetterLevel `default:"3"`
		}{}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Level)
		assert.Equal(t, SetterLevel(103), *got.Level, "the tag sets 3, then the setter adds 100")
	})

	t.Run("not behind a pointer", func(t *testing.T) {
		got := struct {
			Level SetterLevel `default:"3"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, SetterLevel(3), got.Level, "a non-struct value field never reaches a setter")
	})

	t.Run("embedded behind a pointer", func(t *testing.T) {
		got := struct {
			*SetterLevel `default:"3"`
		}{}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.SetterLevel)
		assert.Equal(t, SetterLevel(103), *got.SetterLevel, "the setter adds 100 once")
	})

	t.Run("embedded, not behind a pointer", func(t *testing.T) {
		got := struct {
			SetterLevel `default:"3"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, SetterLevel(3), got.SetterLevel, "no setter, promoted or not")
	})
}

// TestSetter_CalledOnValueReceiver pins that a setter declared on a value receiver is called. The
// pointer's method set lists it as a wrapper the compiler generates, which is also what a promoted
// setter is, so this guards Set's check for a promoted setter against mistaking one for the other.
func TestSetter_CalledOnValueReceiver(t *testing.T) {
	calls := 0
	got := setterByValue{Calls: &calls}

	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 1, calls)
}

// TestSetter_CalledOnGenericStruct pins that a generic type's own setter is called. The compiler
// instantiates a generic method's code itself, so this guards Set's check for a promoted setter,
// which reads where a method's code comes from, against mistaking one for the other.
func TestSetter_CalledOnGenericStruct(t *testing.T) {
	var got setterGeneric[string]

	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, "value", got.Value)
	assert.Equal(t, 1, got.Calls)
}

// TestSetter_RunsAgainOnASecondSet pins that Set is not idempotent as far as setters go: calling it
// twice calls them twice over, which is why a setter has to be idempotent itself.
func TestSetter_RunsAgainOnASecondSet(t *testing.T) {
	var got struct {
		Value SetterCounter
		Ptr   *SetterCounter `default:"{}"`
	}

	require.NoError(t, defaults.Set(&got))
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Ptr)
	assert.Equal(t, 2, got.Value.Calls)
	assert.Equal(t, 2, got.Ptr.Calls)
}
