package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// NestEmbedded is embedded in nestLeaf, so the containers below also cover an embedded struct
// reached through them. Exported because an embedded field takes the name of its type, and an
// unexported one would be beyond reflect's reach.
type NestEmbedded struct {
	Depth int `default:"1"`
}

// nestLeaf is the element type of the nested containers below. It is package-level only so the
// tests can share one shape; it carries no method.
type nestLeaf struct {
	NestEmbedded `default:"{}"`

	Name string `default:"leaf"`
	Kept int
}

func TestSet_DeepSliceOfStructs(t *testing.T) {
	type sample struct {
		Deep [][][]nestLeaf
	}

	got := sample{Deep: [][][]nestLeaf{{{{Kept: 123}}}}}
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, [][][]nestLeaf{{{{NestEmbedded: NestEmbedded{Depth: 1}, Name: "leaf", Kept: 123}}}}, got.Deep)
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
		"outer": {"inner": {NestEmbedded: NestEmbedded{Depth: 1}, Name: "leaf", Kept: 123}},
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
		{"key": {{NestEmbedded: NestEmbedded{Depth: 1}, Name: "leaf", Kept: 123}}},
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
	assert.Equal(t, nestLeaf{NestEmbedded: NestEmbedded{Depth: 1}, Name: "leaf", Kept: 123}, *got.Deep["key"][0])
}

// TestSet_DeepContainersFromTag covers nesting created entirely by a tag, rather than by the caller.
func TestSet_DeepContainersFromTag(t *testing.T) {
	type sample struct {
		Deep map[string][]nestLeaf `default:"{\"key\": [{\"Kept\": 123}]}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, map[string][]nestLeaf{
		"key": {{NestEmbedded: NestEmbedded{Depth: 1}, Name: "leaf", Kept: 123}},
	}, got.Deep)
}
