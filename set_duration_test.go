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
// duration string on a plain int64 is its nanosecond count. The README states both, under
// Durations, so a change here is a change there.
//
// Which parser is tried first is deliberately not asserted, because it is not observable: the only
// tag both accept is a zero, which is zero either way, so swapping them changes nothing.
//
// QUIRK: the parsers are picked by kind, not by type, so int64 and time.Duration cannot be told
// apart. That is kept on purpose: matching time.Duration's exact type is the only alternative, and
// it would stop a named duration type from parsing, which TestSet_Duration pins as working. See
// https://github.com/creasty/defaults/issues/66.
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

// TestSet_DurationTagIsTrimmed covers a duration tag carrying surrounding whitespace, which used to
// leave the field at zero. The trim happens where ParseDuration is attempted, so it covers every
// int64-kinded field — including a named duration type, which a check against time.Duration's exact
// type would miss.
func TestSet_DurationTagIsTrimmed(t *testing.T) {
	type myDuration time.Duration
	type sample struct {
		Duration time.Duration `default:" 10s "`
		Named    myDuration    `default:" 10s "`
		Int64    int64         `default:" 1h "`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, 10*time.Second, got.Duration)
	assert.Equal(t, myDuration(10*time.Second), got.Named, "a named duration type is covered too")
	assert.Equal(t, int64(3600000000000), got.Int64, "an int64 takes a padded duration string, as it already took an unpadded one")
}

// TestSet_NumberTagIsNotTrimmed pins the other side of that trim: it belongs to the duration
// attempt alone, so a padded number reaches the numeric parser exactly as written and is rejected.
// It used to be ignored silently, leaving the field at zero.
//
// One field per subtest, because Set returns on the first error: a shared struct would leave the
// second field unexercised.
func TestSet_NumberTagIsNotTrimmed(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		got := struct {
			V int `default:" 1 "`
		}{}

		require.ErrorContains(t, defaults.Set(&got), `field V: invalid default " 1 "`)
		assert.Zero(t, got.V)
	})

	t.Run("int64", func(t *testing.T) {
		got := struct {
			V int64 `default:" 64 "`
		}{}

		require.ErrorContains(t, defaults.Set(&got), `field V: invalid default " 64 "`,
			"the duration attempt trims, its numeric fallback does not")
		assert.Zero(t, got.V)
	})
}
