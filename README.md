### [!!! Looking for a new maintainer !!!](https://github.com/creasty/defaults/issues/61)

defaults
========

[![CI](https://img.shields.io/github/actions/workflow/status/creasty/defaults/ci.yml?branch=master&label=CI)](https://github.com/creasty/defaults/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/creasty/defaults/branch/master/graph/badge.svg)](https://codecov.io/gh/creasty/defaults)
[![GitHub release](https://img.shields.io/github/release/creasty/defaults.svg)](https://github.com/creasty/defaults/releases)
[![License](https://img.shields.io/github/license/creasty/defaults.svg)](./LICENSE)

Initialize structs with default values

- Supports almost all kind of types
  - Scalar types
    - `int/8/16/32/64`, `uint/8/16/32/64`, `float32/64`
    - `uintptr`, `bool`, `string`
  - Complex types
    - `map`, `slice`, `struct`
  - Nested types
    - `map[K1]map[K2]Struct`, `[]map[K1]Struct[]`
  - Aliased types
    - `time.Duration`
    - e.g., `type Enum string`
  - Pointer types
    - e.g., `*SampleStruct`, `*int`
- Recursively initializes fields in a struct
- Dynamically sets default values by:
  - Implementing the [`defaults.Setter`](./setter.go) interface, or
  - Implementing [`encoding.TextUnmarshaler`](https://pkg.go.dev/encoding#TextUnmarshaler), which
    takes precedence over `defaults.Setter` -- the tag is handed to `UnmarshalText` and
    `SetDefaults` is not called
- Preserves non-initial values from being reset with a default value
  - A field is written only while it still holds its type's zero value. No scalar type can tell an
    unspecified value from its zero value — an `int` left alone is `0`, a `string` is `""`, a `bool`
    is `false` — so a zero the caller set on purpose is indistinguishable from one never set, and the
    default replaces it.
  - Use a pointer where that distinction matters. `nil` means unspecified, and a pointer to the zero
    value is preserved: `*bool` is the way to let `false` survive a `default:"true"`.


Usage
-----

```go
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/creasty/defaults"
)

type Gender string

type Sample struct {
	Name    string `default:"John Smith"`
	Age     int    `default:"27"`
	Gender  Gender `default:"m"`
	Working bool   `default:"true"`

	SliceInt    []int    `default:"[1, 2, 3]"`
	SlicePtr    []*int   `default:"[1, 2, 3]"`
	SliceString []string `default:"[\"a\", \"b\"]"`

	MapNull            map[string]int          `default:"{}"`
	Map                map[string]int          `default:"{\"key1\": 123}"`
	MapOfStruct        map[string]OtherStruct  `default:"{\"Key2\": {\"Foo\":123}}"`
	MapOfPtrStruct     map[string]*OtherStruct `default:"{\"Key3\": {\"Foo\":123}}"`
	MapOfStructWithTag map[string]OtherStruct  `default:"{\"Key4\": {\"Foo\":123}}"`

	Struct    OtherStruct  `default:"{\"Foo\": 123}"`
	StructPtr *OtherStruct `default:"{\"Foo\": 123}"`

	NoTag    OtherStruct // Recurses into a nested struct by default
	NoOption OtherStruct `default:"-"` // no option
}

type OtherStruct struct {
	Hello  string `default:"world"` // Tags in a nested struct also work
	Foo    int    `default:"-"`
	Random int    `default:"-"`
}

// SetDefaults implements defaults.Setter interface
func (s *OtherStruct) SetDefaults() {
	if defaults.CanUpdate(s.Random) { // Check if it's a zero value (recommended)
		s.Random = rand.Int() // Set a dynamic value
	}
}

func main() {
	obj := &Sample{}
	if err := defaults.Set(obj); err != nil {
		panic(err)
	}

	out, err := json.MarshalIndent(obj, "", "	")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	// Output:
	// {
	// 	"Name": "John Smith",
	// 	"Age": 27,
	// 	"Gender": "m",
	// 	"Working": true,
	// 	"SliceInt": [
	// 		1,
	// 		2,
	// 		3
	// 	],
	// 	"SlicePtr": [
	// 		1,
	// 		2,
	// 		3
	// 	],
	// 	"SliceString": [
	// 		"a",
	// 		"b"
	// 	],
	// 	"MapNull": {},
	// 	"Map": {
	// 		"key1": 123
	// 	},
	// 	"MapOfStruct": {
	// 		"Key2": {
	// 			"Hello": "world",
	// 			"Foo": 123,
	// 			"Random": 5577006791947779410
	// 		}
	// 	},
	// 	"MapOfPtrStruct": {
	// 		"Key3": {
	// 			"Hello": "world",
	// 			"Foo": 123,
	// 			"Random": 8674665223082153551
	// 		}
	// 	},
	// 	"MapOfStructWithTag": {
	// 		"Key4": {
	// 			"Hello": "world",
	// 			"Foo": 123,
	// 			"Random": 6129484611666145821
	// 		}
	// 	},
	// 	"Struct": {
	// 		"Hello": "world",
	// 		"Foo": 123,
	// 		"Random": 4037200794235010051
	// 	},
	// 	"StructPtr": {
	// 		"Hello": "world",
	// 		"Foo": 123,
	// 		"Random": 3916589616287113937
	// 	},
	// 	"NoTag": {
	// 		"Hello": "world",
	// 		"Foo": 0,
	// 		"Random": 6334824724549167320
	// 	},
	// 	"NoOption": {
	// 		"Hello": "",
	// 		"Foo": 0,
	// 		"Random": 0
	// 	}
	// }
}
```
