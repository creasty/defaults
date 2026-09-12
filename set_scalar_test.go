package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestSet_SignedIntegers(t *testing.T) {
	type sample struct {
		Int   int   `default:"1"`
		Int8  int8  `default:"8"`
		Int16 int16 `default:"16"`
		Int32 int32 `default:"32"`
		Int64 int64 `default:"64"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{Int: 1, Int8: 8, Int16: 16, Int32: 32, Int64: 64}, got)
}

func TestSet_UnsignedIntegers(t *testing.T) {
	type sample struct {
		Uint    uint    `default:"1"`
		Uint8   uint8   `default:"8"`
		Uint16  uint16  `default:"16"`
		Uint32  uint32  `default:"32"`
		Uint64  uint64  `default:"64"`
		Uintptr uintptr `default:"128"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{Uint: 1, Uint8: 8, Uint16: 16, Uint32: 32, Uint64: 64, Uintptr: 128}, got)
}

func TestSet_Floats(t *testing.T) {
	type sample struct {
		Float32 float32 `default:"1.32"`
		Float64 float64 `default:"1.64"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{Float32: 1.32, Float64: 1.64}, got)
}

func TestSet_Bool(t *testing.T) {
	type sample struct {
		True  bool `default:"true"`
		False bool `default:"false"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.True(t, got.True)
	assert.False(t, got.False, "a false default is indistinguishable from the zero value")
}

func TestSet_String(t *testing.T) {
	type sample struct {
		String      string `default:"hello"`
		WithSpaces  string `default:"hello world"`
		LooksNumber string `default:"123"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{String: "hello", WithSpaces: "hello world", LooksNumber: "123"}, got)
}

// TestSet_IntegerLiteralBases covers every width against every base, because the base is a
// parameter of the strconv call made per width: a `ParseInt(v, 10, 16)` slip in one case must fail
// here. Each base uses a literal for the same value, 12, so one whole-struct assertion suffices.
func TestSet_IntegerLiteralBases(t *testing.T) {
	t.Run("octal", func(t *testing.T) {
		type sample struct {
			Int    int    `default:"0o14"`
			Int8   int8   `default:"0o14"`
			Int16  int16  `default:"0o14"`
			Int32  int32  `default:"0o14"`
			Int64  int64  `default:"0o14"`
			Uint   uint   `default:"0o14"`
			Uint8  uint8  `default:"0o14"`
			Uint16 uint16 `default:"0o14"`
			Uint32 uint32 `default:"0o14"`
			Uint64 uint64 `default:"0o14"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{12, 12, 12, 12, 12, 12, 12, 12, 12, 12}, got)
	})

	t.Run("hexadecimal", func(t *testing.T) {
		type sample struct {
			Int    int    `default:"0xc"`
			Int8   int8   `default:"0xc"`
			Int16  int16  `default:"0xc"`
			Int32  int32  `default:"0xc"`
			Int64  int64  `default:"0xc"`
			Uint   uint   `default:"0xc"`
			Uint8  uint8  `default:"0xc"`
			Uint16 uint16 `default:"0xc"`
			Uint32 uint32 `default:"0xc"`
			Uint64 uint64 `default:"0xc"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{12, 12, 12, 12, 12, 12, 12, 12, 12, 12}, got)
	})

	t.Run("binary", func(t *testing.T) {
		type sample struct {
			Int    int    `default:"0b1100"`
			Int8   int8   `default:"0b1100"`
			Int16  int16  `default:"0b1100"`
			Int32  int32  `default:"0b1100"`
			Int64  int64  `default:"0b1100"`
			Uint   uint   `default:"0b1100"`
			Uint8  uint8  `default:"0b1100"`
			Uint16 uint16 `default:"0b1100"`
			Uint32 uint32 `default:"0b1100"`
			Uint64 uint64 `default:"0b1100"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{12, 12, 12, 12, 12, 12, 12, 12, 12, 12}, got)
	})

	t.Run("width boundaries", func(t *testing.T) {
		type sample struct {
			Int8Max  int8   `default:"0x7f"`
			Int8Min  int8   `default:"-0x80"`
			Uint8Max uint8  `default:"0o377"`
			Int16Max int16  `default:"0b111111111111111"`
			Uint64   uint64 `default:"0xffffffffffffffff"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{
			Int8Max:  127,
			Int8Min:  -128,
			Uint8Max: 255,
			Int16Max: 32767,
			Uint64:   18446744073709551615,
		}, got)
	})
}

func TestSet_NamedScalarTypes(t *testing.T) {
	type (
		myInt     int
		myInt8    int8
		myInt16   int16
		myInt32   int32
		myInt64   int64
		myUint    uint
		myUint8   uint8
		myUint16  uint16
		myUint32  uint32
		myUint64  uint64
		myUintptr uintptr
		myFloat32 float32
		myFloat64 float64
		myBool    bool
		myString  string
	)
	type sample struct {
		Int     myInt     `default:"1"`
		Int8    myInt8    `default:"8"`
		Int16   myInt16   `default:"16"`
		Int32   myInt32   `default:"32"`
		Int64   myInt64   `default:"64"`
		Uint    myUint    `default:"1"`
		Uint8   myUint8   `default:"8"`
		Uint16  myUint16  `default:"16"`
		Uint32  myUint32  `default:"32"`
		Uint64  myUint64  `default:"64"`
		Uintptr myUintptr `default:"128"`
		Float32 myFloat32 `default:"1.32"`
		Float64 myFloat64 `default:"1.64"`
		Bool    myBool    `default:"true"`
		String  myString  `default:"hello"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{
		Int: 1, Int8: 8, Int16: 16, Int32: 32, Int64: 64,
		Uint: 1, Uint8: 8, Uint16: 16, Uint32: 32, Uint64: 64, Uintptr: 128,
		Float32: 1.32, Float64: 1.64, Bool: true, String: "hello",
	}, got)
}

// TestSet_UnparsableValuesAreIgnored pins that a value the field's kind cannot parse is dropped
// without a word: the field keeps its zero value and Set reports success.
//
// QUIRK: arguably a malformed tag deserves an error. See
// https://github.com/creasty/defaults/pull/59.
func TestSet_UnparsableValuesAreIgnored(t *testing.T) {
	t.Run("out of range", func(t *testing.T) {
		type sample struct {
			Int8  int8  `default:"999"`
			Uint8 uint8 `default:"256"`
			Uint  uint  `default:"-1"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{}, got)
	})

	t.Run("not a number", func(t *testing.T) {
		type sample struct {
			Int     int     `default:"3.5"`
			Float64 float64 `default:"abc"`
			Uint    uint    `default:"0x"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{}, got)
	})

	t.Run("not a bool", func(t *testing.T) {
		type sample struct {
			Bool bool `default:"notabool"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.False(t, got.Bool)
	})
}

// TestSet_EmptyTagIsNoOp pins that `default:""` is indistinguishable from carrying no tag at all:
// containers and pointers stay nil rather than being allocated empty.
func TestSet_EmptyTagIsNoOp(t *testing.T) {
	type sample struct {
		String string         `default:""`
		Int    int            `default:""`
		Slice  []string       `default:""`
		Map    map[string]int `default:""`
		Ptr    *int           `default:""`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{}, got)
	assert.Nil(t, got.Slice)
	assert.Nil(t, got.Map)
	assert.Nil(t, got.Ptr)
}

// TestSet_UnsupportedKindsAreIgnored pins that kinds setField has no case for are left alone, tag
// or no tag, without an error.
func TestSet_UnsupportedKindsAreIgnored(t *testing.T) {
	type sample struct {
		Iface   interface{} `default:"1"`
		Chan    chan int    `default:"1"`
		Func    func()      `default:"1"`
		Complex complex128  `default:"1"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Nil(t, got.Iface)
	assert.Nil(t, got.Chan)
	assert.Nil(t, got.Func)
	assert.Zero(t, got.Complex)
}

// TestSet_UnexportedFieldsAreSkipped pins that an unexported field is untouched even with a tag
// (reflect cannot set it), and that its presence does not stop its exported siblings.
func TestSet_UnexportedFieldsAreSkipped(t *testing.T) {
	type sample struct {
		Exported   string `default:"set"`
		unexported string `default:"skipped"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, "set", got.Exported)
	assert.Empty(t, got.unexported, "reflect cannot set it, so the tag is inert")
}
