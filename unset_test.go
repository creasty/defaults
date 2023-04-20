package defaults

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUnset(t *testing.T) {
	t.Run("sample unset", func(t *testing.T) {
		s := &Sample{}
		MustSet(s)
		err := Unset(s)
		require.NoError(t, err)
		require.Equal(t, s, &Sample{})
	})

	t.Run("errInvalidType", func(t *testing.T) {
		tmp := 8
		require.Equal(t, Unset(&tmp), errInvalidType)
		require.Panics(t, func() { MustUnset(&tmp) })
	})

	t.Run("not reset by -", func(t *testing.T) {
		s := &struct {
			IgnoreMe string `default:"-" unset:"-"`
		}{
			IgnoreMe: "test",
		}
		MustUnset(s)
		require.Equal(t, s.IgnoreMe, "test")
	})

	t.Run("sampleUnset test", func(t *testing.T) {
		s := &SampleUnset{}
		MustSet(s)
		var testNumPtr *int = nil
		s.SliceOfIntPtrPtr[1] = &testNumPtr
		s.private = StructUnset{WithDefault: "test"}
		MustUnset(s)

		// struct
		require.Equal(t, s.private.WithDefault, "test")
		require.Equal(t, 0, s.Struct.Foo)
		require.Equal(t, 0, s.Struct.Bar)
		require.Nil(t, s.Struct.BarPtr)
		require.Equal(t, 0, *s.Struct.BarPtrWithWalk)
		require.Empty(t, s.StructPtrNoWalk)
		require.Equal(t, "foo", s.StructPtr.EmbeddedUnset.String)
		require.Equal(t, 0, s.StructPtr.EmbeddedUnset.Int)
		require.Equal(t, "foo", s.StructPtr.Struct.String)
		require.Equal(t, 0, s.StructPtr.Struct.Int)

		// slice
		require.Equal(t, 0, s.SliceOfInt[0])
		require.Equal(t, 0, *s.SliceOfIntPtr[0])
		require.Equal(t, (*int)(nil), *s.SliceOfIntPtrPtr[1])
		require.Equal(t, 0, s.SliceOfStruct[0].Foo)
		require.Equal(t, 0, s.SliceOfStructPtr[0].Foo)
		require.Equal(t, 0, s.SliceOfSliceInt[0][0])
		require.Equal(t, 0, *s.SliceOfSliceIntPtr[0][0])
		require.Equal(t, 0, s.SliceOfSliceStruct[0][0].Foo)
		require.Equal(t, 0, s.SliceOfMapOfInt[0]["int1"])
		require.Equal(t, 0, s.SliceOfMapOfStruct[0]["Struct3"].Foo)
		require.Equal(t, 0, s.SliceOfMapOfStructPtr[0]["Struct3"].Foo)
		require.Nil(t, s.SliceSetNil)

		// map
		require.Equal(t, 0, s.MapOfInt["int1"])
		require.Equal(t, 0, *s.MapOfIntPtr["int1"])
		require.Equal(t, 0, s.MapOfStruct["Struct3"].Foo)
		require.Equal(t, 0, s.MapOfStructPtr["Struct3"].Foo)
		require.Equal(t, 0, s.MapOfSliceInt["slice1"][0])
		require.Equal(t, 0, *s.MapOfSliceIntPtr["slice1"][0])
		require.Equal(t, 0, s.MapOfSliceStruct["slice1"][0].Foo)
		require.Equal(t, 0, s.MapOfMapOfInt["map1"]["int1"])
		require.Equal(t, 0, s.MapOfMapOfInt["map1"]["int1"])
		require.Equal(t, 0, s.MapOfMapOfStruct["map1"]["Struct3"].Foo, 0)
		require.Nil(t, s.MapSetNil)

		// map embed
		require.Equal(t, "foo", s.MapOfStruct["Struct3"].String)
		require.Equal(t, "foo", s.MapOfStructPtr["Struct3"].String)
		require.Equal(t, "foo", s.MapOfSliceStruct["slice1"][0].String)
		require.Equal(t, "foo", s.MapOfMapOfStruct["map1"]["Struct3"].String)

	})
}

type EmbeddedUnset struct {
	Int    int    `default:"1"`
	String string `default:"foo" unset:"-"`
}

type StructUnset struct {
	EmbeddedUnset  `default:"{}" unset:"walk"`
	Foo            int           `default:"1"`
	Bar            int           `default:"1"`
	BarPtr         *int          `default:"1"`
	BarPtrWithWalk *int          `default:"1" unset:"walk"`
	WithDefault    string        `default:"foo"`
	Struct         EmbeddedUnset ` unset:"walk"`
}

type SampleUnset struct {
	private         StructUnset  `default:"{}" unset:"walk"`
	Struct          StructUnset  `default:"{}" unset:"walk"`
	StructPtr       *StructUnset `default:"{}" unset:"walk"`
	StructPtrNoWalk *StructUnset `default:"{}"`

	SliceOfInt            []int                     `default:"[1,2,3]" unset:"walk"`
	SliceOfIntPtr         []*int                    `default:"[1,2,3]" unset:"walk"`
	SliceOfIntPtrPtr      []**int                   `default:"[1,2,3]" unset:"walk"`
	SliceOfStruct         []StructUnset             `default:"[{\"Foo\":123}]" unset:"walk"`
	SliceOfStructPtr      []*StructUnset            `default:"[{\"Foo\":123}]" unset:"walk"`
	SliceOfMapOfInt       []map[string]int          `default:"[{\"int1\": 1}]" unset:"walk"`
	SliceOfMapOfStruct    []map[string]StructUnset  `default:"[{\"Struct3\": {\"Foo\":123}}]" unset:"walk"`
	SliceOfMapOfStructPtr []map[string]*StructUnset `default:"[{\"Struct3\": {\"Foo\":123}}]" unset:"walk"`
	SliceOfSliceInt       [][]int                   `default:"[[1,2,3]]" unset:"walk"`
	SliceOfSliceIntPtr    [][]*int                  `default:"[[1,2,3]]" unset:"walk"`
	SliceOfSliceStruct    [][]StructUnset           `default:"[[{\"Foo\":123}]]" unset:"walk"`
	SliceOfSliceStructPtr [][]*StructUnset          `default:"[[{\"Foo\":123}]]" unset:"walk"`
	SliceSetNil           []StructUnset             `default:"[{\"Foo\":123}]"`

	MapOfInt            map[string]int                    `default:"{\"int1\": 1}" unset:"walk"`
	MapOfIntPtr         map[string]*int                   `default:"{\"int1\": 1}" unset:"walk"`
	MapOfStruct         map[string]StructUnset            `default:"{\"Struct3\": {\"Foo\":123}}" unset:"walk" `
	MapOfStructPtr      map[string]*StructUnset           `default:"{\"Struct3\": {\"Foo\":123}}" unset:"walk"`
	MapOfSliceInt       map[string][]int                  `default:"{\"slice1\": [1,2,3]}" unset:"walk"`
	MapOfSliceIntPtr    map[string][]*int                 `default:"{\"slice1\": [1,2,3]}" unset:"walk"`
	MapOfSliceStruct    map[string][]StructUnset          `default:"{\"slice1\": [{\"Foo\":123}]}" unset:"walk"`
	MapOfSliceStructPtr map[string][]*StructUnset         `default:"{\"slice1\": [{\"Foo\":123}]}" unset:"walk"`
	MapOfMapOfInt       map[string]map[string]int         `default:"{\"map1\": {\"int1\": 1}}" unset:"walk"`
	MapOfMapOfStruct    map[string]map[string]StructUnset `default:"{\"map1\": {\"Struct3\": {\"Foo\":123}}}" unset:"walk"`

	MapSetNil map[string]StructUnset `default:"{\"Struct3\": {\"Foo\":123}}"`
}
