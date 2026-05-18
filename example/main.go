package main

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/creasty/defaults"
)

type Gender string

type Sample struct {
	Name    string `default:"John Smith" default_alt:"Jane Doe"`
	Age     int    `default:"27" default_alt:"28"`
	Gender  Gender `default:"m" default_alt:"f"`
	Working bool   `default:"true"`

	SliceInt    []int    `default:"[1, 2, 3]" default_alt:"[4, 5, 6]"`
	SlicePtr    []*int   `default:"[1, 2, 3]" default_alt:"[4, 5, 6]"`
	SliceString []string `default:"[\"a\", \"b\"]" default_alt:"[\"c\", \"d\"]"`

	MapNull            map[string]int          `default:"{}"` // Using default_alt would leave as a nil map
	Map                map[string]int          `default:"{\"key1\": 123}" default_alt:"{\"key1\": 456}"`
	MapOfStruct        map[string]OtherStruct  `default:"{\"Key2\": {\"Foo\":123}}" default_alt:"{\"Key2\": {\"Foo\":456}}"`
	MapOfPtrStruct     map[string]*OtherStruct `default:"{\"Key3\": {\"Foo\":123}}" default_alt:"{\"Key3\": {\"Foo\":456}}"`
	MapOfStructWithTag map[string]OtherStruct  `default:"{\"Key4\": {\"Foo\":123}}" default_alt:"{\"Key4\": {\"Foo\":456}}"`

	Struct    OtherStruct  `default:"{\"Foo\": 123}" default:"{\"Foo\": 456}"`
	StructPtr *OtherStruct `default:"{\"Foo\": 123}" default:"{\"Foo\": 456}"`

	NoTag    OtherStruct // Recurses into a nested struct by default
	NoOption OtherStruct `default:"-"` // no option
}

type OtherStruct struct {
	Hello  string `default:"world" default_alt:"person"` // Tags in a nested struct also work
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
	/*
	{
		"Name": "John Smith",
		"Age": 27,
		"Gender": "m",
		"Working": true,
		"SliceInt": [
			1,
			2,
			3
		],
		"SlicePtr": [
			1,
			2,
			3
		],
		"SliceString": [
			"a",
			"b"
		],
		"MapNull": {},
		"Map": {
			"key1": 123
		},
		"MapOfStruct": {
			"Key2": {
				"Hello": "world",
				"Foo": 123,
				"Random": 7924310560672650386
			}
		},
		"MapOfPtrStruct": {
			"Key3": {
				"Hello": "world",
				"Foo": 123,
				"Random": 5864827752208870454
			}
		},
		"MapOfStructWithTag": {
			"Key4": {
				"Hello": "world",
				"Foo": 123,
				"Random": 9020930397672710767
			}
		},
		"Struct": {
			"Hello": "world",
			"Foo": 123,
			"Random": 2100809313953071154
		},
		"StructPtr": {
			"Hello": "world",
			"Foo": 123,
			"Random": 4589638854976076769
		},
		"NoTag": {
			"Hello": "world",
			"Foo": 0,
			"Random": 93481895446446669
		},
		"NoOption": {
			"Hello": "",
			"Foo": 0,
			"Random": 0
		}
	}
	*/

	// Or with a specific tag name:
	otherObj := &Sample{}
	if err = defaults.SetWithTag(otherObj, "default_alt"); err != nil {
		panic(err)
	}

	out, err = json.MarshalIndent(otherObj, "", "	")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	// Output:
	/*
	{
		"Name": "Jane Doe",
		"Age": 28,
		"Gender": "f",
		"Working": false,
		"SliceInt": [
			4,
			5,
			6
		],
		"SlicePtr": [
			4,
			5,
			6
		],
		"SliceString": [
			"c",
			"d"
		],
		"MapNull": null,
		"Map": {
			"key1": 456
		},
		"MapOfStruct": {
			"Key2": {
				"Hello": "person",
				"Foo": 456,
				"Random": 3527321973631593020
			}
		},
		"MapOfPtrStruct": {
			"Key3": {
				"Hello": "person",
				"Foo": 456,
				"Random": 6391170103701304261
			}
		},
		"MapOfStructWithTag": {
			"Key4": {
				"Hello": "person",
				"Foo": 456,
				"Random": 5779747168002044569
			}
		},
		"Struct": {
			"Hello": "person",
			"Foo": 0,
			"Random": 4391597250420769051
		},
		"StructPtr": null,
		"NoTag": {
			"Hello": "person",
			"Foo": 0,
			"Random": 5000048969203402981
		},
		"NoOption": {
			"Hello": "person",
			"Foo": 0,
			"Random": 4800395242796521920
		}
	}
	*/
}
