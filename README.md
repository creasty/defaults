defaults
========

[![Build Status](https://travis-ci.org/creasty/defaults.svg?branch=master)](https://travis-ci.org/creasty/defaults)
[![codecov](https://codecov.io/gh/creasty/defaults/branch/master/graph/badge.svg)](https://codecov.io/gh/creasty/defaults)

Initialize members in struct with default values


Usage
-----

```go
obj := &SampleStruct{}
if err := defaults.SetDefaults(obj); err != nil {
	panic(err)
}
```


Supported types
---------------

- Scalar types
  - `int`, `int8`, `int16`, `int32`, `int64`
  - `uint`, `uint8`, `uint16`, `uint32`, `uint64`
  - `uintptr`
  - `float32`, `float64`
  - `bool`
  - `string`
  - `time.Duration`
- Complex types
  - `map` and `slice`
  - `struct`
- [`defaults.Defaulter`](./defaults_setter.go) interface


Take a look at [defaults_test.go](./defaults_test.go).

```go
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
}

type Struct struct {
	Foo         int
	Bar         int
	WithDefault string `default:"foo"`
}

// SetDefaults implements defaults.Defaulter interface
func (s *Struct) SetDefaults() {
	s.Bar = 456
}
```
