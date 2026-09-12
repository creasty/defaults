package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// TestSet_UnexportedScalarFieldIsSkipped pins that an unexported field is untouched even with a tag
// (reflect cannot set it), and that its presence does not stop its exported siblings.
func TestSet_UnexportedScalarFieldIsSkipped(t *testing.T) {
	type sample struct {
		Exported   string `default:"set"`
		unexported string `default:"skipped"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, "set", got.Exported)
	assert.Empty(t, got.unexported, "reflect cannot set it, so the tag is inert")
}

// TestSet_UnexportedCompositeFieldsAreSkippedWholesale pins how far the CanSet guard reaches: an
// unexported field is skipped whatever its type, so a whole subtree of tags below one goes
// unapplied. Only the scalar case is obvious; these are not.
func TestSet_UnexportedCompositeFieldsAreSkippedWholesale(t *testing.T) {
	type inner struct {
		Name string `default:"inner"`
	}
	type sample struct {
		hidden    inner
		hiddenPtr *inner `default:"{}"`
		hiddenMap map[string]inner
		hiddenSl  []inner
	}

	got := sample{
		hiddenMap: map[string]inner{"a": {}},
		hiddenSl:  []inner{{}},
	}
	require.NoError(t, defaults.Set(&got))

	assert.Empty(t, got.hidden.Name, "a nested tag below an unexported field never applies")
	assert.Nil(t, got.hiddenPtr, "not even an explicit tag allocates it")
	assert.Empty(t, got.hiddenMap["a"].Name)
	assert.Empty(t, got.hiddenSl[0].Name)
}
