package unmarshal_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults/internal/unmarshal"
)

// The fixtures record what they were offered, so a test can tell an unmarshaler that was never asked
// from one that was asked and took it, and each carries the error it refuses with, so a test can name
// the rejection it expects back. They are package-level because each carries a method, and named with
// a file-unique prefix.

// unmarshalText takes every value it is offered, and keeps it.
type unmarshalText struct{ text string }

func (u *unmarshalText) UnmarshalText(text []byte) error {
	u.text = string(text)
	return nil
}

// unmarshalJSON takes every value it is offered, and keeps it.
type unmarshalJSON struct{ data string }

func (u *unmarshalJSON) UnmarshalJSON(data []byte) error {
	u.data = string(data)
	return nil
}

// unmarshalBoth implements both, keeping what each was offered.
type unmarshalBoth struct {
	text string
	data string
}

func (u *unmarshalBoth) UnmarshalText(text []byte) error {
	u.text = string(text)
	return nil
}

func (u *unmarshalBoth) UnmarshalJSON(data []byte) error {
	u.data = string(data)
	return nil
}

// unmarshalTextRejects refuses every value from UnmarshalText, and takes it by UnmarshalJSON.
type unmarshalTextRejects struct {
	refusal error
	data    string
}

func (u *unmarshalTextRejects) UnmarshalText([]byte) error { return u.refusal }

func (u *unmarshalTextRejects) UnmarshalJSON(data []byte) error {
	u.data = string(data)
	return nil
}

// unmarshalBothReject refuses every value from either, with an error of its own each.
type unmarshalBothReject struct {
	textRefusal error
	jsonRefusal error
}

func (u *unmarshalBothReject) UnmarshalText([]byte) error { return u.textRefusal }

func (u *unmarshalBothReject) UnmarshalJSON([]byte) error { return u.jsonRefusal }

// unmarshalJSONRejects implements only UnmarshalJSON, and refuses.
type unmarshalJSONRejects struct{ refusal error }

func (u *unmarshalJSONRejects) UnmarshalJSON([]byte) error { return u.refusal }

// unmarshalNeither implements neither.
type unmarshalNeither struct{}

// TestTag_Taken covers a value one of the target's unmarshalers takes, which is the whole point of
// offering it: the caller parses nothing itself when Tag reports true.
func TestTag_Taken(t *testing.T) {
	t.Run("by UnmarshalText", func(t *testing.T) {
		target := &unmarshalText{}

		taken, err := unmarshal.Tag(target, "8080")

		assert.True(t, taken)
		require.NoError(t, err)
		assert.Equal(t, "8080", target.text)
	})

	t.Run("by UnmarshalJSON, with no UnmarshalText to ask first", func(t *testing.T) {
		target := &unmarshalJSON{}

		taken, err := unmarshal.Tag(target, "8080")

		assert.True(t, taken)
		require.NoError(t, err)
		assert.Equal(t, "8080", target.data)
	})

	t.Run("by UnmarshalJSON, after UnmarshalText refused it", func(t *testing.T) {
		target := &unmarshalTextRejects{refusal: errors.New("no")}

		taken, err := unmarshal.Tag(target, "8080")

		assert.True(t, taken)
		require.NoError(t, err)
		assert.Equal(t, "8080", target.data)
	})
}

// TestTag_TextIsAskedFirst covers the order: a target that implements both is asked through
// UnmarshalText, and UnmarshalJSON is never offered the value.
func TestTag_TextIsAskedFirst(t *testing.T) {
	target := &unmarshalBoth{}

	taken, err := unmarshal.Tag(target, "8080")

	assert.True(t, taken)
	require.NoError(t, err)
	assert.Equal(t, "8080", target.text)
	assert.Empty(t, target.data, "UnmarshalJSON is not offered a value UnmarshalText took")
}

// TestTag_Rejected covers what the caller reports when neither took the value: UnmarshalText's
// rejection, since it was asked first, and UnmarshalJSON's when there was no UnmarshalText to ask.
func TestTag_Rejected(t *testing.T) {
	t.Run("both refuse, so the first rejection is the error", func(t *testing.T) {
		target := &unmarshalBothReject{textRefusal: errors.New("text"), jsonRefusal: errors.New("json")}

		taken, err := unmarshal.Tag(target, "8080")

		assert.False(t, taken)
		assert.Same(t, target.textRefusal, err)
	})

	t.Run("only UnmarshalJSON is there, and refuses", func(t *testing.T) {
		target := &unmarshalJSONRejects{refusal: errors.New("json")}

		taken, err := unmarshal.Tag(target, "8080")

		assert.False(t, taken)
		assert.Same(t, target.refusal, err)
	})
}

// TestTag_NotOffered covers the values an unmarshaler is not asked about at all, which leave the
// target as it was and report no error: there is nothing to report when nothing was asked.
func TestTag_NotOffered(t *testing.T) {
	t.Run("an empty value, to either", func(t *testing.T) {
		target := &unmarshalBoth{}

		taken, err := unmarshal.Tag(target, "")

		assert.False(t, taken)
		require.NoError(t, err)
		assert.Empty(t, target.text)
		assert.Empty(t, target.data)
	})

	t.Run("an empty object or array, to UnmarshalJSON", func(t *testing.T) {
		for _, value := range []string{"{}", "[]"} {
			target := &unmarshalJSON{}

			taken, err := unmarshal.Tag(target, value)

			assert.False(t, taken, value)
			require.NoError(t, err, value)
			assert.Empty(t, target.data, value)
		}
	})

	t.Run("an empty object or array, which UnmarshalText is still offered", func(t *testing.T) {
		for _, value := range []string{"{}", "[]"} {
			target := &unmarshalText{}

			taken, err := unmarshal.Tag(target, value)

			assert.True(t, taken, value)
			require.NoError(t, err, value)
			assert.Equal(t, value, target.text, value)
		}
	})

	t.Run("anything, to a target that implements neither", func(t *testing.T) {
		taken, err := unmarshal.Tag(&unmarshalNeither{}, "8080")

		assert.False(t, taken)
		assert.NoError(t, err)
	})
}
