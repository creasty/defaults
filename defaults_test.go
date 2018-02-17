package defaults

import (
	"reflect"
	"testing"
	"time"

	"github.com/creasty/defaults/internal/fixture"
)

type (
	MyInt     int
	MyInt8    int8
	MyInt16   int16
	MyInt32   int32
	MyInt64   int64
	MyUint    uint
	MyUint8   uint8
	MyUint16  uint16
	MyUint32  uint32
	MyUint64  uint64
	MyUintptr uintptr
	MyFloat32 float32
	MyFloat64 float64
	MyBool    bool
	MyString  string
	MyMap     map[string]int
	MySlice   []int
)

type Sample struct {
	Int       int           `default:"1"`
	Int8      int8          `default:"8"`
	Int16     int16         `default:"16"`
	Int32     int32         `default:"32"`
	Int64     int64         `default:"64"`
	Uint      uint          `default:"1"`
	Uint8     uint8         `default:"8"`
	Uint16    uint16        `default:"16"`
	Uint32    uint32        `default:"32"`
	Uint64    uint64        `default:"64"`
	Uintptr   uintptr       `default:"1"`
	Float32   float32       `default:"1.32"`
	Float64   float64       `default:"1.64"`
	BoolTrue  bool          `default:"true"`
	BoolFalse bool          `default:"false"`
	String    string        `default:"hello"`
	Duration  time.Duration `default:"10s"`

	Struct    Struct         `default:"{}"`
	StructPtr *Struct        `default:"{}"`
	Map       map[string]int `default:"{}"`
	Slice     []string       `default:"[]"`

	MyInt       MyInt     `default:"1"`
	MyInt8      MyInt8    `default:"8"`
	MyInt16     MyInt16   `default:"16"`
	MyInt32     MyInt32   `default:"32"`
	MyInt64     MyInt64   `default:"64"`
	MyUint      MyUint    `default:"1"`
	MyUint8     MyUint8   `default:"8"`
	MyUint16    MyUint16  `default:"16"`
	MyUint32    MyUint32  `default:"32"`
	MyUint64    MyUint64  `default:"64"`
	MyUintptr   MyUintptr `default:"1"`
	MyFloat32   MyFloat32 `default:"1.32"`
	MyFloat64   MyFloat64 `default:"1.64"`
	MyBoolTrue  MyBool    `default:"true"`
	MyBoolFalse MyBool    `default:"false"`
	MyString    MyString  `default:"hello"`
	MyMap       MyMap     `default:"{}"`
	MySlice     MySlice   `default:"[]"`

	StructWithJSON    Struct         `default:"{\"Foo\": 123}"`
	StructPtrWithJSON *Struct        `default:"{\"Foo\": 123}"`
	MapWithJSON       map[string]int `default:"{\"foo\": 123}"`
	SliceWithJSON     []string       `default:"[\"foo\"]"`

	Empty string `default:""`

	NoDefault       *string `default:"-"`
	NoDefaultStruct Struct  `default:"-"`

	MapWithNoTag       map[string]int
	SliceWithNoTag     []string
	StructPtrWithNoTag *Struct
	StructWithNoTag    Struct

	NonInitialString    string  `default:"foo"`
	NonInitialSlice     []int   `default:"[123]"`
	NonInitialStruct    Struct  `default:"{}"`
	NonInitialStructPtr *Struct `default:"{}"`
}

type Struct struct {
	Emmbeded `default:"{}"`

	Foo         int
	Bar         int
	WithDefault string `default:"foo"`
}

func (s *Struct) SetDefaults() {
	s.Bar = 456
}

type Emmbeded struct {
	Int int `default:"1"`
}

func TestInit(t *testing.T) {
	sample := &Sample{
		NonInitialString:    "string",
		NonInitialSlice:     []int{1, 2, 3},
		NonInitialStruct:    Struct{Foo: 123},
		NonInitialStructPtr: &Struct{Foo: 123},
	}

	if err := Set(sample); err != nil {
		t.Fatalf("it should return an error: %v", err)
	}

	if err := Set(1); err == nil {
		t.Fatalf("it should return an error when used for a non-pointer type")
	}

	Set(&fixture.Sample{}) // should not panic

	t.Run("scalar types", func(t *testing.T) {
		if sample.Int != 1 {
			t.Errorf("it should initialize int")
		}
		if sample.Int8 != 8 {
			t.Errorf("it should initialize int8")
		}
		if sample.Int16 != 16 {
			t.Errorf("it should initialize int16")
		}
		if sample.Int32 != 32 {
			t.Errorf("it should initialize int32")
		}
		if sample.Int64 != 64 {
			t.Errorf("it should initialize int64")
		}
		if sample.Uint != 1 {
			t.Errorf("it should initialize uint")
		}
		if sample.Uint8 != 8 {
			t.Errorf("it should initialize uint8")
		}
		if sample.Uint16 != 16 {
			t.Errorf("it should initialize uint16")
		}
		if sample.Uint32 != 32 {
			t.Errorf("it should initialize uint32")
		}
		if sample.Uint64 != 64 {
			t.Errorf("it should initialize uint64")
		}
		if sample.Uintptr != 1 {
			t.Errorf("it should initialize uintptr")
		}
		if sample.Float32 != 1.32 {
			t.Errorf("it should initialize float32")
		}
		if sample.Float64 != 1.64 {
			t.Errorf("it should initialize float64")
		}
		if sample.BoolTrue != true {
			t.Errorf("it should initialize bool (true)")
		}
		if sample.BoolFalse != false {
			t.Errorf("it should initialize bool (false)")
		}
		if sample.String != "hello" {
			t.Errorf("it should initialize string")
		}
	})

	t.Run("complex types", func(t *testing.T) {
		if sample.StructPtr == nil {
			t.Errorf("it should initialize struct pointer")
		}
		if sample.Map == nil {
			t.Errorf("it should initialize map")
		}
		if sample.Slice == nil {
			t.Errorf("it should initialize slice")
		}
	})

	t.Run("aliased types", func(t *testing.T) {
		if sample.MyInt != 1 {
			t.Errorf("it should initialize int")
		}
		if sample.MyInt8 != 8 {
			t.Errorf("it should initialize int8")
		}
		if sample.MyInt16 != 16 {
			t.Errorf("it should initialize int16")
		}
		if sample.MyInt32 != 32 {
			t.Errorf("it should initialize int32")
		}
		if sample.MyInt64 != 64 {
			t.Errorf("it should initialize int64")
		}
		if sample.MyUint != 1 {
			t.Errorf("it should initialize uint")
		}
		if sample.MyUint8 != 8 {
			t.Errorf("it should initialize uint8")
		}
		if sample.MyUint16 != 16 {
			t.Errorf("it should initialize uint16")
		}
		if sample.MyUint32 != 32 {
			t.Errorf("it should initialize uint32")
		}
		if sample.MyUint64 != 64 {
			t.Errorf("it should initialize uint64")
		}
		if sample.MyUintptr != 1 {
			t.Errorf("it should initialize uintptr")
		}
		if sample.MyFloat32 != 1.32 {
			t.Errorf("it should initialize float32")
		}
		if sample.MyFloat64 != 1.64 {
			t.Errorf("it should initialize float64")
		}
		if sample.MyBoolTrue != true {
			t.Errorf("it should initialize bool (true)")
		}
		if sample.MyBoolFalse != false {
			t.Errorf("it should initialize bool (false)")
		}
		if sample.MyString != "hello" {
			t.Errorf("it should initialize string")
		}

		if sample.MyMap == nil {
			t.Errorf("it should initialize map")
		}
		if sample.MySlice == nil {
			t.Errorf("it should initialize slice")
		}
	})

	t.Run("nested", func(t *testing.T) {
		if sample.Struct.WithDefault != "foo" {
			t.Errorf("it should set default on inner field in struct")
		}
		if sample.StructPtr == nil || sample.StructPtr.WithDefault != "foo" {
			t.Errorf("it should set default on inner field in struct pointer")
		}
		if sample.Struct.Emmbeded.Int != 1 {
			t.Errorf("it should set default on an emmbeded struct")
		}
	})

	t.Run("complex types with json", func(t *testing.T) {
		if sample.StructWithJSON.Foo != 123 {
			t.Errorf("it should initialize struct with json")
		}
		if sample.StructPtrWithJSON == nil || sample.StructPtrWithJSON.Foo != 123 {
			t.Errorf("it should initialize struct pointer with json")
		}
		if sample.MapWithJSON["foo"] != 123 {
			t.Errorf("it should initialize map with json")
		}
		if len(sample.SliceWithJSON) == 0 || sample.SliceWithJSON[0] != "foo" {
			t.Errorf("it should initialize slice with json")
		}
	})

	t.Run("Setter interface", func(t *testing.T) {
		if sample.Struct.Bar != 456 {
			t.Errorf("it should initialize struct")
		}
		if sample.StructPtr == nil || sample.StructPtr.Bar != 456 {
			t.Errorf("it should initialize struct pointer")
		}
	})

	t.Run("non-initial value", func(t *testing.T) {
		if sample.NonInitialString != "string" {
			t.Errorf("it should not override non-initial value")
		}
		if !reflect.DeepEqual(sample.NonInitialSlice, []int{1, 2, 3}) {
			t.Errorf("it should not override non-initial value")
		}
		if !reflect.DeepEqual(sample.NonInitialStruct, Struct{Emmbeded: Emmbeded{Int: 1}, Foo: 123, Bar: 456, WithDefault: "foo"}) {
			t.Errorf("it should not override non-initial value but set defaults for fields")
		}
		if !reflect.DeepEqual(sample.NonInitialStructPtr, &Struct{Emmbeded: Emmbeded{Int: 1}, Foo: 123, Bar: 456, WithDefault: "foo"}) {
			t.Errorf("it should not override non-initial value but set defaults for fields")
		}
	})

	t.Run("no tag", func(t *testing.T) {
		if sample.MapWithNoTag != nil {
			t.Errorf("it should not initialize pointer type (map)")
		}
		if sample.SliceWithNoTag != nil {
			t.Errorf("it should not initialize pointer type (slice)")
		}
		if sample.StructPtrWithNoTag != nil {
			t.Errorf("it should not initialize pointer type (struct)")
		}
		if sample.StructWithNoTag.WithDefault != "foo" {
			t.Errorf("it should automatically recurse into a struct even without a tag")
		}
	})

	t.Run("opt-out", func(t *testing.T) {
		if sample.NoDefault != nil {
			t.Errorf("it should not be set")
		}
		if sample.NoDefaultStruct.WithDefault != "" {
			t.Errorf("it should not initialize a struct with default values")
		}
	})
}

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
