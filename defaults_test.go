package defaults

import (
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

	Struct    struct{}        `default:"{}"`
	StructPtr *struct{}       `default:"{}"`
	Map       map[string]bool `default:"{}"`
	Slice     []string        `default:"{}"`

	Empty     string `default:""`
	NoDefault string
}

func TestInit(t *testing.T) {
	sample := new(Sample)

	if err := Init(sample); err != nil {
		t.Fatalf("it should return an error: %v", err)
	}

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
	if sample.StructPtr == nil {
		t.Errorf("it should initialize struct")
	}
	if sample.Map == nil {
		t.Errorf("it should initialize map")
	}
	if sample.Slice == nil {
		t.Errorf("it should initialize slice")
	}
}
