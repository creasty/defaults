package defaults

import (
	"reflect"
	"testing"
	"time"
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

	StructWithJSON    Struct         `default:"{\"Foo\": 123}"`
	StructPtrWithJSON *Struct        `default:"{\"Foo\": 123}"`
	MapWithJSON       map[string]int `default:"{\"foo\": 123}"`
	SliceWithJSON     []string       `default:"[\"foo\"]"`

	Empty     string `default:""`
	NoDefault string

	NonInitialString string `default:"foo"`
	NonInitialSlice  []int  `default:"[123]"`
	NonInitialStruct Struct `default:"{}"`
}

type Struct struct {
	Foo         int
	Bar         int
	WithDefault string `default:"foo"`
}

func (s *Struct) SetDefaults() {
	s.Bar = 456
}

func TestInit(t *testing.T) {
	sample := &Sample{
		NonInitialString: "string",
		NonInitialSlice:  []int{1, 2, 3},
		NonInitialStruct: Struct{Foo: 123},
	}

	if err := SetDefaults(sample); err != nil {
		t.Fatalf("it should return an error: %v", err)
	}

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

	t.Run("nested", func(t *testing.T) {
		if sample.Struct.WithDefault != "foo" {
			t.Errorf("it should set default on inner field in struct")
		}
		if sample.StructPtr == nil || sample.StructPtr.WithDefault != "foo" {
			t.Errorf("it should set default on inner field in struct pointer")
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
		if !reflect.DeepEqual(sample.NonInitialStruct, Struct{Foo: 123}) {
			t.Errorf("it should not override non-initial value")
		}
	})
}
