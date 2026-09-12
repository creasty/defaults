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

// TestSet_DurationIsTriedBeforeInteger pins that every int64-kinded field goes through
// time.ParseDuration first and strconv.ParseInt only as a fallback. So a bare number on a Duration
// is nanoseconds, and a duration string on a plain int64 is its nanosecond count.
//
// QUIRK: the parsers are picked by kind, not by type, so int64 and time.Duration are
// indistinguishable here. See https://github.com/creasty/defaults/pull/55.
func TestSet_DurationIsTriedBeforeInteger(t *testing.T) {
	type sample struct {
		BareNumberOnDuration time.Duration `default:"1"`
		DurationOnInt64      int64         `default:"1h"`
		NumberOnInt64        int64         `default:"64"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, time.Nanosecond, got.BareNumberOnDuration, "a bare number is parsed as nanoseconds")
	assert.Equal(t, int64(3600000000000), got.DurationOnInt64, "a duration string lands on a plain int64")
	assert.Equal(t, int64(64), got.NumberOnInt64, "ParseInt still handles what ParseDuration rejects")
}

// TestSet_DurationDoesNotAffectNarrowerIntegers pins that the duration fallback is int64-only:
// narrower widths reject a duration string and keep their zero value.
func TestSet_DurationDoesNotAffectNarrowerIntegers(t *testing.T) {
	type sample struct {
		Int   int   `default:"1h"`
		Int32 int32 `default:"1h"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, sample{}, got)
}
