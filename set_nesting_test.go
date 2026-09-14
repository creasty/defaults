package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// TestSet_NestedContainers covers containers nested inside one another, whether the caller built them
// or a tag did: the structs at the bottom get their defaults, and so does a struct embedded in them.
func TestSet_NestedContainers(t *testing.T) {
	// Embedded is exported because an embedded field takes the name of its type, and an unexported
	// one would be skipped wholesale, as TestSet_UnexportedCompositeFieldsAreSkippedWholesale pins.
	type Embedded struct {
		Depth int `default:"1"`
	}
	type leaf struct {
		Embedded `default:"{}"`

		Name string `default:"leaf"`
		Kept int
	}

	filled := leaf{Embedded: Embedded{Depth: 1}, Name: "leaf", Kept: 123}

	t.Run("slice of slice of slice", func(t *testing.T) {
		got := struct {
			Deep [][][]leaf
		}{Deep: [][][]leaf{{{{Kept: 123}}}}}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, [][][]leaf{{{filled}}}, got.Deep)
	})

	t.Run("map of map", func(t *testing.T) {
		got := struct {
			Deep map[string]map[string]leaf
		}{Deep: map[string]map[string]leaf{
			"outer": {"inner": {Kept: 123}},
		}}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, map[string]map[string]leaf{"outer": {"inner": filled}}, got.Deep)
	})

	t.Run("slice of map of slice", func(t *testing.T) {
		got := struct {
			Deep []map[string][]leaf
		}{Deep: []map[string][]leaf{
			{"key": {{Kept: 123}}},
		}}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, []map[string][]leaf{{"key": {filled}}}, got.Deep)
	})

	t.Run("map of slice of pointers", func(t *testing.T) {
		got := struct {
			Deep map[string][]*leaf
		}{Deep: map[string][]*leaf{
			"key": {{Kept: 123}},
		}}

		require.NoError(t, defaults.Set(&got))

		require.Len(t, got.Deep["key"], 1)
		require.NotNil(t, got.Deep["key"][0])
		assert.Equal(t, filled, *got.Deep["key"][0])
	})

	t.Run("created by a tag", func(t *testing.T) {
		got := struct {
			Deep map[string][]leaf `default:"{\"key\": [{\"Kept\": 123}]}"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, map[string][]leaf{"key": {filled}}, got.Deep)
	})
}
