package defaults_test

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/creasty/defaults"
)

// Every case calls b.ReportAllocs. A case that calls Set does its setup and checks the value it
// measures once before b.ResetTimer, and then does the same work on every iteration: it resets a zero
// value in place and fills it again, or it runs Set again on a value Set leaves as it is. A CanUpdate
// case has no setup, so it checks inside its loop.
//
// `make bench-compare` builds this file alone against other commits, so it depends only on the
// package's API. Every case builds and passes its checks on every commit from v1.10.0 on, so no input
// is one those commits cannot handle, such as a cycle.

// BenchmarkSet measures whole calls, the rows pull requests have reported. Their loops are kept as
// they were when those numbers were taken, the allocation of a fresh `var got st` included.
func BenchmarkSet(b *testing.B) {
	b.Run("scalars", func(b *testing.B) {
		type st struct {
			Str  string        `default:"str"`
			Int  int           `default:"1"`
			Bool bool          `default:"true"`
			Dur  time.Duration `default:"5s"`
		}

		var check st
		if err := defaults.Set(&check); err != nil || check.Str != "str" || check.Int != 1 || !check.Bool || check.Dur != 5*time.Second {
			b.Fatalf("Set = %v, %+v", err, check)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var got st
			if err := defaults.Set(&got); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("composites", func(b *testing.B) {
		type inner struct {
			Name string `default:"inner"`
			Port int    `default:"8080"`
		}
		type st struct {
			Inner inner
			Ptr   *inner            `default:"{}"`
			Slice []string          `default:"[\"a\",\"b\"]"`
			Map   map[string]string `default:"{\"k\":\"v\"}"`
		}

		var check st
		if err := defaults.Set(&check); err != nil || check.Inner.Port != 8080 || check.Ptr == nil || check.Ptr.Name != "inner" || len(check.Slice) != 2 || check.Map["k"] != "v" {
			b.Fatalf("Set = %v, %+v", err, check)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var got st
			if err := defaults.Set(&got); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCanUpdate measures the zero check on its own, which is what Set spends per field.
func BenchmarkCanUpdate(b *testing.B) {
	b.Run("scalar", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !defaults.CanUpdate(0) {
				b.Fatal("a zero int is updatable")
			}
		}
	})

	b.Run("struct", func(b *testing.B) {
		type st struct {
			Str   string
			Int   int
			Slice []string
			Map   map[string]int
		}

		var zero st

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !defaults.CanUpdate(zero) {
				b.Fatal("a zero struct is updatable")
			}
		}
	})

	// The struct case's four fields, but comparable, so reflect compares the whole struct against zero
	// memory at once instead of checking field by field.
	b.Run("comparable_struct", func(b *testing.B) {
		type st struct {
			Str string
			Int int
			Ptr *string
			Dur time.Duration
		}

		var zero st

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !defaults.CanUpdate(zero) {
				b.Fatal("a zero struct is updatable")
			}
		}
	})
}

// BenchmarkParse applies one tag to one zero field. unparsed_kind is the cost every other row
// includes: a tagged zero field with nothing parsed and nothing written.
func BenchmarkParse(b *testing.B) {
	b.Run("unparsed_kind", func(b *testing.B) {
		type st struct {
			V complex128 `default:"1"`
		}
		benchFill(b, func(got *st) bool { return got.V == 0 })
	})

	b.Run("string", func(b *testing.B) {
		type st struct {
			V string `default:"item"`
		}
		benchFill(b, func(got *st) bool { return got.V == "item" })
	})

	b.Run("int", func(b *testing.B) {
		type st struct {
			V int `default:"8080"`
		}
		benchFill(b, func(got *st) bool { return got.V == 8080 })
	})

	// A plain number on an int64 fails time.ParseDuration before strconv.ParseInt takes it.
	b.Run("int64", func(b *testing.B) {
		type st struct {
			V int64 `default:"8080"`
		}
		benchFill(b, func(got *st) bool { return got.V == 8080 })
	})

	b.Run("duration", func(b *testing.B) {
		type st struct {
			V time.Duration `default:"5s"`
		}
		benchFill(b, func(got *st) bool { return got.V == 5*time.Second })
	})

	// One field per kind the switch parses, each value under 256.
	b.Run("kinds", func(b *testing.B) {
		type st struct {
			Bool    bool    `default:"true"`
			Int     int     `default:"1"`
			Int8    int8    `default:"8"`
			Int16   int16   `default:"16"`
			Int32   int32   `default:"32"`
			Int64   int64   `default:"64"`
			Uint    uint    `default:"1"`
			Uint8   uint8   `default:"8"`
			Uint16  uint16  `default:"16"`
			Uint32  uint32  `default:"32"`
			Uint64  uint64  `default:"64"`
			Uintptr uintptr `default:"128"`
			Float32 float32 `default:"1.5"`
			Float64 float64 `default:"1.5"`
			String  string  `default:"s"`
		}
		benchFill(b, func(got *st) bool {
			return got.Bool && got.Int64 == 64 && got.Uintptr == 128 && got.Float32 == 1.5 && got.String == "s"
		})
	})

	// Both int64 parsers fail on the empty tag, which is let through without joining their errors.
	b.Run("empty_tag", func(b *testing.B) {
		type st struct {
			V time.Duration `default:""`
		}
		benchFill(b, func(got *st) bool { return got.V == 0 })
	})

	// A tag value with escaped quotes, as every JSON string has, which StructTag.Lookup unquotes.
	b.Run("escaped_tag/bytes=1024", func(b *testing.B) {
		value := strings.Repeat(`"key":1,`, 128)
		typ := benchStruct(1, reflect.TypeOf(""), reflect.StructTag("default:"+strconv.Quote(value)))
		benchFillValue(b, typ, func(v reflect.Value) bool { return v.Field(0).String() == value })
	})

	b.Run("json_slice", func(b *testing.B) {
		type st struct {
			V []int `default:"[1,2,3]"`
		}
		benchFill(b, func(got *st) bool { return len(got.V) == 3 && got.V[2] == 3 })
	})

	b.Run("json_map", func(b *testing.B) {
		type st struct {
			V map[string]int `default:"{\"a\":1,\"b\":2}"`
		}
		benchFill(b, func(got *st) bool { return len(got.V) == 2 && got.V["b"] == 2 })
	})

	// The tag is decoded into the field itself, and the walk then goes on below it: A came from the
	// tag, and B is filled from its own default.
	b.Run("json_struct", func(b *testing.B) {
		type leaf struct {
			Name string `default:"leaf"`
		}
		type mid struct {
			A, B leaf
		}
		type st struct {
			M mid `default:"{\"A\":{\"Name\":\"x\"}}"`
		}
		benchFill(b, func(got *st) bool { return got.M.A.Name == "x" && got.M.B.Name == "leaf" })
	})

	// The unmarshalers store the tag's length, 4, so the check tells them apart from parsing by kind,
	// which would give 8080.
	b.Run("unmarshaler/text", func(b *testing.B) {
		type st struct {
			V benchText `default:"8080"`
		}
		benchFill(b, func(got *st) bool { return got.V == 4 })
	})

	b.Run("unmarshaler/json", func(b *testing.B) {
		type st struct {
			V benchJSON `default:"8080"`
		}
		benchFill(b, func(got *st) bool { return got.V == 4 })
	})

	b.Run("unmarshaler/rejected", func(b *testing.B) {
		type st struct {
			V benchRejecting `default:"8080"`
		}
		benchFill(b, func(got *st) bool { return got.V == 8080 })
	})
}

// BenchmarkWalk measures going through a value with little or nothing left to fill. Unless a case
// says otherwise, it runs Set again on a value the first Set filled.
func BenchmarkWalk(b *testing.B) {
	type item struct {
		Name string `default:"item"`
		Port int    `default:"8080"`
	}
	filled := item{Name: "item", Port: 8080}

	for _, n := range []int{1, 64} {
		b.Run("struct/untagged/fields="+strconv.Itoa(n), func(b *testing.B) {
			v := reflect.New(benchStruct(n, reflect.TypeOf(0), ""))
			benchRepeat(b, v.Interface(), func() bool { return v.Elem().NumField() == n })
		})
	}

	b.Run("struct/foreign_tags/fields=64", func(b *testing.B) {
		v := reflect.New(benchStruct(64, reflect.TypeOf(0), `json:"field_name,omitempty" yaml:"field_name" validate:"required"`))
		benchRepeat(b, v.Interface(), func() bool { return v.Elem().NumField() == 64 })
	})

	for _, depth := range []int{4, 64} {
		b.Run("struct/nested_values/depth="+strconv.Itoa(depth), func(b *testing.B) {
			v := reflect.New(benchNestedValues(depth))
			benchRepeat(b, v.Interface(), func() bool {
				e := v.Elem()
				for i := 0; i < depth; i++ {
					e = e.Field(0)
				}
				return e.Field(0).String() == "x"
			})
		})
	}

	// The tagged pointers are set back to nil on every iteration, so each level is allocated again.
	for _, depth := range []int{4, 64} {
		b.Run("pointer/tagged_chain/depth="+strconv.Itoa(depth), func(b *testing.B) {
			benchFillValue(b, benchTaggedChain(depth), func(e reflect.Value) bool {
				for i := 0; i < depth; i++ {
					e = e.Field(0).Elem()
				}
				return e.Field(0).Int() == 1
			})
		})
	}

	// A linked list the caller built, its pointers untagged: the path grows to 1000 structs below the one
	// Set is handed, and Set checks each struct it enters against the path above it.
	b.Run("pointer/caller_chain/depth=1000", func(b *testing.B) {
		type link struct {
			Name string `default:"item"`
			Next *link
		}
		head := &link{}
		for i := 0; i < 1000; i++ {
			head = &link{Next: head}
		}
		benchRepeat(b, head, func() bool {
			n, depth := head, 0
			for ; n.Next != nil; n = n.Next {
				depth++
			}
			return depth == 1000 && head.Name == "item" && n.Name == "item"
		})
	})

	// The tagged pointers are set back to nil on every iteration, so each is allocated and parsed again.
	b.Run("pointer/tagged_ints/fields=10", func(b *testing.B) {
		typ := benchStruct(10, reflect.TypeOf((*int)(nil)), `default:"1"`)
		benchFillValue(b, typ, func(v reflect.Value) bool { return v.Field(9).Elem().Int() == 1 })
	})

	// tagged_ints' struct, but re-Set: the pointers the first Set allocated now point at filled ints, as
	// a caller's *int or *bool does.
	b.Run("pointer/to_ints/fields=10", func(b *testing.B) {
		v := reflect.New(benchStruct(10, reflect.TypeOf((*int)(nil)), `default:"1"`))
		benchRepeat(b, v.Interface(), func() bool { return v.Elem().Field(9).Elem().Int() == 1 })
	})

	b.Run("pointer/to_nil_slices/fields=10", func(b *testing.B) {
		v := reflect.New(benchStruct(10, reflect.TypeOf((*[]string)(nil)), ""))
		for i := 0; i < 10; i++ {
			v.Elem().Field(i).Set(reflect.ValueOf(new([]string)))
		}
		benchRepeat(b, v.Interface(), func() bool {
			f := v.Elem().Field(9)
			return !f.IsNil() && f.Elem().IsNil()
		})
	})

	// The elements are filled before the first Set, so the value is the same whether or not Set follows
	// the pointer.
	b.Run("pointer/to_filled_slice/elements=1000", func(b *testing.B) {
		items := make([]item, 1000)
		for i := range items {
			items[i] = filled
		}
		got := struct{ Items *[]item }{Items: &items}
		benchRepeat(b, &got, func() bool { return (*got.Items)[999] == filled })
	})

	// Eight levels of nodes whose two pointers share the level below: 256 paths to one item, and no
	// cycle.
	b.Run("pointer/shared/depth=8", func(b *testing.B) {
		type node struct {
			L, R *node
			Leaf *item
		}
		root := &node{Leaf: &item{}}
		for i := 0; i < 8; i++ {
			root = &node{L: root, R: root}
		}
		benchRepeat(b, root, func() bool {
			n := root
			for n.L != nil {
				n = n.L
			}
			return root.L == root.R && *n.Leaf == filled
		})
	})

	b.Run("slice/structs/elements=1000", func(b *testing.B) {
		got := struct{ Items []item }{Items: make([]item, 1000)}
		benchRepeat(b, &got, func() bool { return got.Items[0] == filled && got.Items[999] == filled })
	})

	b.Run("slice/structs_with_pointer/elements=1000", func(b *testing.B) {
		type ref struct {
			Name string `default:"item"`
			Port int    `default:"8080"`
			Ref  *int
		}
		got := struct{ Items []ref }{Items: make([]ref, 1000)}
		benchRepeat(b, &got, func() bool { return got.Items[999] == ref{Name: "item", Port: 8080} })
	})

	b.Run("slice/structs_with_setter/elements=1000", func(b *testing.B) {
		got := struct{ Items []benchSetterItem }{Items: make([]benchSetterItem, 1000)}
		benchRepeat(b, &got, func() bool { return got.Items[999] == benchSetterItem{Name: "item", Port: 8080} })
	})

	b.Run("slice/pointers/elements=1000", func(b *testing.B) {
		got := struct{ Items []*item }{Items: make([]*item, 1000)}
		for i := range got.Items {
			got.Items[i] = &item{}
		}
		benchRepeat(b, &got, func() bool { return *got.Items[0] == filled && *got.Items[999] == filled })
	})

	b.Run("slice/ints/elements=1000", func(b *testing.B) {
		got := struct{ Ints []int }{Ints: make([]int, 1000)}
		for i := range got.Ints {
			got.Ints[i] = i
		}
		benchRepeat(b, &got, func() bool { return got.Ints[999] == 999 })
	})

	b.Run("map/structs/entries=100", func(b *testing.B) {
		got := struct{ M map[string]item }{M: make(map[string]item, 100)}
		for i := 0; i < 100; i++ {
			got.M[strconv.Itoa(i)] = item{}
		}
		benchRepeat(b, &got, func() bool { return got.M["0"] == filled && got.M["99"] == filled })
	})

	b.Run("map/zero_structs/entries=100", func(b *testing.B) {
		keys := make([]string, 100)
		got := struct{ M map[string]item }{M: make(map[string]item, 100)}
		for i := range keys {
			keys[i] = strconv.Itoa(i)
			got.M[keys[i]] = item{}
		}
		if err := defaults.Set(&got); err != nil || got.M["0"] != filled || got.M["99"] != filled {
			b.Fatalf("Set = %v, %+v", err, got.M["99"])
		}

		// The values are zeroed inside the timer: 100 map assignments, the same on every iteration.
		// b.StopTimer around them would cost more than they do.
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, k := range keys {
				got.M[k] = item{}
			}
			if err := defaults.Set(&got); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Debug holds false, the zero value its tag asks for, so every Set parses the tag again and writes
	// the same value into each copy.
	b.Run("map/structs_reparsed/entries=100", func(b *testing.B) {
		type flagged struct {
			Name  string `default:"item"`
			Port  int    `default:"8080"`
			Debug bool   `default:"false"`
		}
		got := struct{ M map[string]flagged }{M: make(map[string]flagged, 100)}
		for i := 0; i < 100; i++ {
			got.M[strconv.Itoa(i)] = flagged{}
		}
		benchRepeat(b, &got, func() bool { return got.M["99"] == flagged{Name: "item", Port: 8080} })
	})

	b.Run("map/pointers/entries=100", func(b *testing.B) {
		got := struct{ M map[string]*item }{M: make(map[string]*item, 100)}
		for i := 0; i < 100; i++ {
			got.M[strconv.Itoa(i)] = &item{}
		}
		benchRepeat(b, &got, func() bool { return *got.M["0"] == filled && *got.M["99"] == filled })
	})

	// Each slice is copied out of the map and walked like a struct value, though its one element is
	// shared with the map's own slice, and the copy is not stored back.
	b.Run("map/slices/entries=100", func(b *testing.B) {
		got := struct{ M map[string][]item }{M: make(map[string][]item, 100)}
		for i := 0; i < 100; i++ {
			got.M[strconv.Itoa(i)] = make([]item, 1)
		}
		benchRepeat(b, &got, func() bool { return got.M["0"][0] == filled && got.M["99"][0] == filled })
	})

	b.Run("map/ints/entries=1000", func(b *testing.B) {
		got := struct{ M map[string]int }{M: make(map[string]int, 1000)}
		for i := 0; i < 1000; i++ {
			got.M[strconv.Itoa(i)] = i
		}
		benchRepeat(b, &got, func() bool { return got.M["999"] == 999 })
	})
}

// BenchmarkFail measures a Set that fails. The first error is checked, so each case measures the
// failure it names.
func BenchmarkFail(b *testing.B) {
	b.Run("field", func(b *testing.B) {
		type st struct {
			Port int `default:"http"`
		}
		benchFail[st](b, func(err error) bool {
			return strings.HasPrefix(err.Error(), `field Port: invalid default "http": `)
		})
	})

	// The failure is four zero values down: a struct, a pointer its tag allocates, the struct behind
	// it, and a field after one that was filled.
	b.Run("nested", func(b *testing.B) {
		type leaf struct {
			Name string `default:"leaf"`
			Port int    `default:"http"`
		}
		type mid struct {
			Leaf *leaf `default:"{}"`
		}
		type st struct {
			Mid mid
		}
		benchFail[st](b, func(err error) bool {
			return strings.HasPrefix(err.Error(), `field Port: invalid default "http": `)
		})
	})

	// UnmarshalText rejects the tag, encoding/json fails on it too, and the rejection is reported.
	b.Run("unmarshaler", func(b *testing.B) {
		type st struct {
			V benchRejectingStruct `default:"nonsense"`
		}
		benchFail[st](b, func(err error) bool {
			return strings.HasPrefix(err.Error(), `field V: invalid default "nonsense": `) && errors.Is(err, strconv.ErrSyntax)
		})
	})
}

// benchFill measures filling a zero T. The value is allocated once and reset in place before each
// Set, so allocs/op counts Set's allocations and not the value's. filled checks the first Set.
func benchFill[T any](b *testing.B, filled func(*T) bool) {
	b.Helper()

	got := new(T)
	if err := defaults.Set(got); err != nil {
		b.Fatal(err)
	}
	if !filled(got) {
		b.Fatalf("Set left %+v", *got)
	}

	var zero T
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*got = zero
		if err := defaults.Set(got); err != nil {
			b.Fatal(err)
		}
	}
}

// benchFillValue is benchFill for a type built at run time.
func benchFillValue(b *testing.B, typ reflect.Type, filled func(reflect.Value) bool) {
	b.Helper()

	v := reflect.New(typ)
	ptr, elem := v.Interface(), v.Elem()
	if err := defaults.Set(ptr); err != nil {
		b.Fatal(err)
	}
	if !filled(elem) {
		b.Fatalf("Set left %+v", elem)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elem.SetZero()
		if err := defaults.Set(ptr); err != nil {
			b.Fatal(err)
		}
	}
}

// benchRepeat measures Set on a value it leaves as it is. The first Set fills ptr, and ready checks
// that the value is in the state the case measures.
func benchRepeat(b *testing.B, ptr interface{}, ready func() bool) {
	b.Helper()

	if err := defaults.Set(ptr); err != nil {
		b.Fatal(err)
	}
	if !ready() {
		b.Fatalf("Set left %+v", ptr)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := defaults.Set(ptr); err != nil {
			b.Fatal(err)
		}
	}
}

// benchFail measures a Set on a zero T that fails. valid checks the first error.
func benchFail[T any](b *testing.B, valid func(error) bool) {
	b.Helper()

	got := new(T)
	if err := defaults.Set(got); err == nil || !valid(err) {
		b.Fatalf("Set = %v, want the error this case measures", err)
	}

	var zero T
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*got = zero
		if defaults.Set(got) == nil {
			b.Fatal("Set succeeded")
		}
	}
}

// benchStruct builds a struct of n fields of typ, F0 to Fn-1, each tagged tag: a field count or a tag
// chosen at run time, where a literal struct would take a line per field.
func benchStruct(n int, typ reflect.Type, tag reflect.StructTag) reflect.Type {
	fields := make([]reflect.StructField, n)
	for i := range fields {
		fields[i] = reflect.StructField{Name: "F" + strconv.Itoa(i), Type: typ, Tag: tag}
	}
	return reflect.StructOf(fields)
}

// benchNestedValues builds depth structs nested by value, each holding the next as its first field.
// Only the innermost has a default. A []string at every level makes every level non-comparable, so
// reflect's zero check walks down to the innermost before it finds a field that is not zero.
func benchNestedValues(depth int) reflect.Type {
	tags := reflect.StructField{Name: "Tags", Type: reflect.TypeOf([]string(nil))}
	t := reflect.StructOf([]reflect.StructField{
		{Name: "Name", Type: reflect.TypeOf(""), Tag: `default:"x"`},
		tags,
	})
	for i := 0; i < depth; i++ {
		t = reflect.StructOf([]reflect.StructField{{Name: "Next", Type: t}, tags})
	}
	return t
}

// benchTaggedChain builds depth structs linked by pointers tagged `default:"{}"`, ending in a struct
// whose int has a default. Every level is a type of its own, so the check for a default that recurses
// without end never fires.
func benchTaggedChain(depth int) reflect.Type {
	t := reflect.StructOf([]reflect.StructField{{Name: "V", Type: reflect.TypeOf(0), Tag: `default:"1"`}})
	for i := 0; i < depth; i++ {
		t = reflect.StructOf([]reflect.StructField{{Name: "Next", Type: reflect.PointerTo(t), Tag: `default:"{}"`}})
	}
	return t
}

// benchText takes every tag and stores its length.
type benchText int

func (t *benchText) UnmarshalText(text []byte) error {
	*t = benchText(len(text))
	return nil
}

// benchJSON takes every tag and stores its length.
type benchJSON int

func (t *benchJSON) UnmarshalJSON(data []byte) error {
	*t = benchJSON(len(data))
	return nil
}

// benchRejecting rejects every tag from both unmarshalers, so parsing by kind takes it. The rejection
// is an error that already exists, so it allocates nothing of its own.
type benchRejecting int

func (*benchRejecting) UnmarshalText([]byte) error { return strconv.ErrSyntax }

func (*benchRejecting) UnmarshalJSON([]byte) error { return strconv.ErrSyntax }

// benchRejectingStruct rejects every tag, and is a struct, so encoding/json is tried next.
type benchRejectingStruct struct {
	V int
}

func (*benchRejectingStruct) UnmarshalText([]byte) error { return strconv.ErrSyntax }

// benchSetterItem has a SetDefaults that does nothing, so Set leaves a filled one as it is.
type benchSetterItem struct {
	Name string `default:"item"`
	Port int    `default:"8080"`
}

func (*benchSetterItem) SetDefaults() {}
