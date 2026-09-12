package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestSet_StructTagEmptyObject(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Struct inner `default:"{}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, inner{Name: "inner"}, got.Struct)
}

func TestSet_StructTagJSON(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
		Foo  int
	}
	type sample struct {
		Struct inner `default:"{\"Foo\": 123}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, inner{Name: "inner", Foo: 123}, got.Struct, "the tag fills Foo, the nested tag fills Name")
}

// TestSet_StructTagJSONWins covers precedence: what the parent's JSON provides is not overwritten by
// the child's own default.
func TestSet_StructTagJSONWins(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		Struct inner `default:"{\"Name\": \"from parent\"}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, "from parent", got.Struct.Name)
}

func TestSet_UntaggedStructRecurses(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		Struct inner
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, inner{Name: "inner"}, got.Struct, "a struct is descended into without a tag")
}

// TestSet_EmbeddedStruct covers promoted fields. The embedded types are named with an initial
// capital on purpose: the embedded field takes the type's name, and an unexported field name would
// make it unsettable through reflect.
func TestSet_EmbeddedStruct(t *testing.T) {
	type Tagged struct {
		Int int `default:"1"`
	}
	type Untagged struct {
		String string `default:"hello"`
	}
	type sample struct {
		Tagged `default:"{}"`
		Untagged
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 1, got.Tagged.Int)
	assert.Equal(t, "hello", got.Untagged.String, "an embedded struct is descended into without a tag")
}

func TestSet_EmbeddedPointerStruct(t *testing.T) {
	type Inner struct {
		Int int `default:"1"`
	}
	type sample struct {
		*Inner `default:"{}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.Inner)
	assert.Equal(t, 1, got.Int, "the embedded pointer is allocated and its fields promoted")
}

func TestSet_EmbeddedPointerStructWithoutTagStaysNil(t *testing.T) {
	type Inner struct {
		Int int `default:"1"`
	}
	type sample struct {
		*Inner
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Nil(t, got.Inner, "an embedded pointer follows the same rule as any other nil pointer")
}
