package defaults_test

import (
	"math"
	"strconv"
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
			Int     int     `default:"0o14"`
			Int8    int8    `default:"0o14"`
			Int16   int16   `default:"0o14"`
			Int32   int32   `default:"0o14"`
			Int64   int64   `default:"0o14"`
			Uint    uint    `default:"0o14"`
			Uint8   uint8   `default:"0o14"`
			Uint16  uint16  `default:"0o14"`
			Uint32  uint32  `default:"0o14"`
			Uint64  uint64  `default:"0o14"`
			Uintptr uintptr `default:"0o14"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12}, got)
	})

	t.Run("hexadecimal", func(t *testing.T) {
		type sample struct {
			Int     int     `default:"0xc"`
			Int8    int8    `default:"0xc"`
			Int16   int16   `default:"0xc"`
			Int32   int32   `default:"0xc"`
			Int64   int64   `default:"0xc"`
			Uint    uint    `default:"0xc"`
			Uint8   uint8   `default:"0xc"`
			Uint16  uint16  `default:"0xc"`
			Uint32  uint32  `default:"0xc"`
			Uint64  uint64  `default:"0xc"`
			Uintptr uintptr `default:"0xc"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12}, got)
	})

	t.Run("binary", func(t *testing.T) {
		type sample struct {
			Int     int     `default:"0b1100"`
			Int8    int8    `default:"0b1100"`
			Int16   int16   `default:"0b1100"`
			Int32   int32   `default:"0b1100"`
			Int64   int64   `default:"0b1100"`
			Uint    uint    `default:"0b1100"`
			Uint8   uint8   `default:"0b1100"`
			Uint16  uint16  `default:"0b1100"`
			Uint32  uint32  `default:"0b1100"`
			Uint64  uint64  `default:"0b1100"`
			Uintptr uintptr `default:"0b1100"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12}, got)
	})
}

// TestSet_NumericWidthBoundaries pins the bit size each numeric case parses with. Every case passes
// that width to strconv as a literal, and a wrong one is invisible except at the boundary: the
// largest value that must parse, and a value past it that must not.
//
// The out-of-range values are chosen to truncate to something non-zero, which is what makes the
// second subtest able to fail at all. A power of two truncates to zero, so `uint8 default:"256"`
// asserting the zero value would still pass if the width were widened to 16; 257 truncates to 1 and
// would not.
func TestSet_NumericWidthBoundaries(t *testing.T) {
	t.Run("largest value parses", func(t *testing.T) {
		type sample struct {
			Int8    int8    `default:"127"`
			Int16   int16   `default:"32767"`
			Int32   int32   `default:"2147483647"`
			Int64   int64   `default:"9223372036854775807"`
			Uint8   uint8   `default:"255"`
			Uint16  uint16  `default:"65535"`
			Uint32  uint32  `default:"4294967295"`
			Uint64  uint64  `default:"18446744073709551615"`
			Float32 float32 `default:"3.4028234e38"`
			Float64 float64 `default:"1.7976931348623157e308"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{
			Int8:    math.MaxInt8,
			Int16:   math.MaxInt16,
			Int32:   math.MaxInt32,
			Int64:   math.MaxInt64,
			Uint8:   math.MaxUint8,
			Uint16:  math.MaxUint16,
			Uint32:  math.MaxUint32,
			Uint64:  math.MaxUint64,
			Float32: math.MaxFloat32,
			Float64: math.MaxFloat64,
		}, got)
	})

	t.Run("smallest value parses", func(t *testing.T) {
		type sample struct {
			Int8  int8  `default:"-128"`
			Int16 int16 `default:"-32768"`
			Int32 int32 `default:"-2147483648"`
			Int64 int64 `default:"-9223372036854775808"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{
			Int8:  math.MinInt8,
			Int16: math.MinInt16,
			Int32: math.MinInt32,
			Int64: math.MinInt64,
		}, got)
	})

	t.Run("past the boundary the field is left alone", func(t *testing.T) {
		type sample struct {
			Int8    int8    `default:"128"`        // -128 if parsed any wider
			Int8Min int8    `default:"-129"`       // 127 if parsed any wider
			Int16   int16   `default:"32768"`      // -32768
			Int32   int32   `default:"2147483648"` // -2147483648
			Uint8   uint8   `default:"257"`        // 1
			Uint16  uint16  `default:"65537"`      // 1
			Uint32  uint32  `default:"4294967297"` // 1
			Float32 float32 `default:"1e39"`       // +Inf
			Float64 float64 `default:"1e309"`      // +Inf
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{}, got)
	})

	// The widths of int, uint and uintptr come from strconv.IntSize, so their boundary literals are
	// only meaningful where that is 64.
	t.Run("platform-width integers", func(t *testing.T) {
		if strconv.IntSize != 64 {
			t.Skipf("strconv.IntSize is %d; the literals below are 64-bit", strconv.IntSize)
		}

		type sample struct {
			Int     int     `default:"9223372036854775807"`
			Uint    uint    `default:"18446744073709551615"`
			Uintptr uintptr `default:"18446744073709551615"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, sample{
			Int:     math.MaxInt,
			Uint:    math.MaxUint,
			Uintptr: math.MaxUint,
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

// TestSet_EmptyTag covers `default:""`, which is a request for the zero value rather than an
// absence: Set reads the tag with Tag.Lookup, so an empty tag differs from no tag at all.
//
// Every kind treats it the same way — a scalar is set to its zero value, which is
// indistinguishable from doing nothing, and a pointer, slice and map are each allocated empty.
// TestSet_UntaggedPointerStaysNil, TestSet_UntaggedSliceStaysNil and TestSet_UntaggedMapStaysNil
// cover the other side, where no tag means no allocation.
func TestSet_EmptyTag(t *testing.T) {
	type sample struct {
		String string         `default:""`
		Int    int            `default:""`
		Slice  []string       `default:""`
		Map    map[string]int `default:""`
		Ptr    *int           `default:""`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Empty(t, got.String)
	assert.Zero(t, got.Int)

	require.NotNil(t, got.Slice)
	assert.Empty(t, got.Slice)
	require.NotNil(t, got.Map)
	assert.Empty(t, got.Map)
	require.NotNil(t, got.Ptr)
	assert.Zero(t, *got.Ptr)
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

// TestSet_InterfaceFieldsAreNotFollowed pins that a non-nil interface is not descended into, even
// when it holds a pointer to a struct carrying tags. The same value in a typed field would be filled
// in, so what a field can receive depends on how it is declared.
func TestSet_InterfaceFieldsAreNotFollowed(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		Iface interface{}
	}

	got := sample{Iface: &inner{}}
	require.NoError(t, defaults.Set(&got))

	held, ok := got.Iface.(*inner)
	require.True(t, ok)
	assert.Empty(t, held.Name, "the tag inside the held value never applies")
}
