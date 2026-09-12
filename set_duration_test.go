package defaults_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

func TestSet_Duration(t *testing.T) {
	type myDuration time.Duration
	type sample struct {
		Simple   time.Duration `default:"10s"`
		Compound time.Duration `default:"1h30m"`
		Named    myDuration    `default:"10s"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{
		Simple:   10 * time.Second,
		Compound: 90 * time.Minute,
		Named:    myDuration(10 * time.Second),
	}, got)
}

// TestSet_DurationAndIntegerShareOneParser pins the outcomes of int64-kinded fields being handed to
// both time.ParseDuration and strconv.ParseInt: a bare number on a Duration is nanoseconds, and a
// duration string on a plain int64 is its nanosecond count.
//
// Which parser is tried first is deliberately not asserted, because it is not observable: the two
// accept disjoint inputs, so swapping them changes nothing.
//
// QUIRK: the parsers are picked by kind, not by type, so int64 and time.Duration cannot be told
// apart. See https://github.com/creasty/defaults/issues/66.
func TestSet_DurationAndIntegerShareOneParser(t *testing.T) {
	type sample struct {
		BareNumberOnDuration time.Duration `default:"1"`
		DurationOnInt64      int64         `default:"1h"`
		NumberOnInt64        int64         `default:"64"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, time.Nanosecond, got.BareNumberOnDuration, "a bare number is nanoseconds")
	assert.Equal(t, int64(3600000000000), got.DurationOnInt64, "a duration string lands on a plain int64")
	assert.Equal(t, int64(64), got.NumberOnInt64, "and a plain number still works")
}

// TestSet_DurationStringOnANarrowerIntegerIsRejected pins that the duration fallback is int64-only.
// A narrower width rejects a duration string outright, where it used to keep its zero value and
// report success.
func TestSet_DurationStringOnANarrowerIntegerIsRejected(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		got := struct {
			V int `default:"1h"`
		}{}

		require.Error(t, defaults.Set(&got), "only int64 gets the duration fallback")
	})

	t.Run("int32", func(t *testing.T) {
		got := struct {
			V int32 `default:"1h"`
		}{}

		require.Error(t, defaults.Set(&got))
	})
}
