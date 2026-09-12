package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// nestLeaf is the element type of the nested containers below. It is package-level only so the
// three tests can share one shape; it carries no method.
type nestLeaf struct {
	Name string `default:"leaf"`
	Kept int
}

func TestSet_DeepSliceOfStructs(t *testing.T) {
	type sample struct {
		Deep [][][]nestLeaf
	}

	got := sample{Deep: [][][]nestLeaf{{{{Kept: 123}}}}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, [][][]nestLeaf{{{{Name: "leaf", Kept: 123}}}}, got.Deep)
}

func TestSet_MapOfMapOfStructs(t *testing.T) {
	type sample struct {
		Deep map[string]map[string]nestLeaf
	}

	got := sample{Deep: map[string]map[string]nestLeaf{
		"outer": {"inner": {Kept: 123}},
	}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, map[string]map[string]nestLeaf{
		"outer": {"inner": {Name: "leaf", Kept: 123}},
	}, got.Deep)
}

func TestSet_SliceOfMapOfSliceOfStructs(t *testing.T) {
	type sample struct {
		Deep []map[string][]nestLeaf
	}

	got := sample{Deep: []map[string][]nestLeaf{
		{"key": {{Kept: 123}}},
	}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, []map[string][]nestLeaf{
		{"key": {{Name: "leaf", Kept: 123}}},
	}, got.Deep)
}

func TestSet_MapOfSliceOfPointerStructs(t *testing.T) {
	type sample struct {
		Deep map[string][]*nestLeaf
	}

	got := sample{Deep: map[string][]*nestLeaf{
		"key": {{Kept: 123}},
	}}
	require.NoError(t, defaults.Set(&got))

	require.Len(t, got.Deep["key"], 1)
	require.NotNil(t, got.Deep["key"][0])
	assert.Equal(t, nestLeaf{Name: "leaf", Kept: 123}, *got.Deep["key"][0])
}

// TestSet_DeepContainersFromTag covers nesting created entirely by a tag, rather than by the caller.
func TestSet_DeepContainersFromTag(t *testing.T) {
	type sample struct {
		Deep map[string][]nestLeaf `default:"{\"key\": [{\"Kept\": 123}]}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, map[string][]nestLeaf{
		"key": {{Name: "leaf", Kept: 123}},
	}, got.Deep)
}
