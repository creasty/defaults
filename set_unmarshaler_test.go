package defaults_test

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/creasty/defaults"
)

// umJSONEnum implements json.Unmarshaler only, so it exercises the JSON branch of the
// unmarshal-by-interface path. All types in this file are package-level because they need methods.
type umJSONEnum int

func (e *umJSONEnum) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if i, err := strconv.Atoi(s); err == nil {
		*e = umJSONEnum(i)
		return nil
	}
	if s == "one" {
		*e = 1
		return nil
	}
	return errors.New("unknown enum")
}

// umJSONRecorder implements only json.Unmarshaler, and records the raw bytes it was handed so a
// test can tell whether it was called at all.
type umJSONRecorder struct {
	Raw string
}

func (u *umJSONRecorder) UnmarshalJSON(b []byte) error {
	u.Raw = string(b)
	return nil
}

// umBoth implements both interfaces and records which one was used.
type umBoth struct {
	Via string
}

func (u *umBoth) UnmarshalText([]byte) error {
	u.Via = "text"
	return nil
}

func (u *umBoth) UnmarshalJSON([]byte) error {
	u.Via = "json"
	return nil
}

// umFailingText always fails, so Set has to fall back to parsing by kind.
type umFailingText string

func (u *umFailingText) UnmarshalText([]byte) error {
	return errors.New("always fails")
}

// umFailingJSON always fails, so Set has to fall back to parsing by kind.
type umFailingJSON int

func (u *umFailingJSON) UnmarshalJSON([]byte) error {
	return errors.New("always fails")
}

func TestSet_TextUnmarshaler(t *testing.T) {
	type sample struct {
		IP net.IP `default:"10.0.0.1"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.True(t, got.IP.Equal(net.ParseIP("10.0.0.1")), "got %v", got.IP)
}

func TestSet_TextUnmarshalerPointer(t *testing.T) {
	type sample struct {
		IP *net.IP `default:"10.0.0.1"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	require.NotNil(t, got.IP)
	assert.True(t, got.IP.Equal(net.ParseIP("10.0.0.1")), "got %v", got.IP)
}

func TestSet_JSONUnmarshaler(t *testing.T) {
	type sample struct {
		FromName   umJSONEnum `default:"\"one\""`
		FromNumber umJSONEnum `default:"\"2\""`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, umJSONEnum(1), got.FromName)
	assert.Equal(t, umJSONEnum(2), got.FromNumber)
}

// TestSet_TextUnmarshalerWinsOverJSON pins the precedence between the two interfaces.
func TestSet_TextUnmarshalerWinsOverJSON(t *testing.T) {
	type sample struct {
		Both umBoth `default:"{\"Via\": \"from json\"}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, "text", got.Both.Via)
}

// TestSet_FailingUnmarshalerFallsBackToKind covers what happens when the type's own unmarshaler
// rejects the value: the error is not reported, and the value is parsed by kind instead.
func TestSet_FailingUnmarshalerFallsBackToKind(t *testing.T) {
	type sample struct {
		Text umFailingText `default:"hello"`
		JSON umFailingJSON `default:"5"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, umFailingText("hello"), got.Text, "parsed as a string after UnmarshalText failed")
	assert.Equal(t, umFailingJSON(5), got.JSON, "parsed as an int after UnmarshalJSON failed")
}

// TestSet_EmptyContainerTagsAndJSONUnmarshaler pins which empty-container tags reach a custom
// UnmarshalJSON. The interface path withholds both `{}` and `[]`, on the grounds that they mean
// "allocate an empty one" — but for a struct-kinded type the kind-based path then hands `[]` to
// encoding/json anyway, which calls the very same method. So the two are not symmetric.
//
// This needs a type that implements *only* json.Unmarshaler. With one that also implements
// UnmarshalText, text wins first and the JSON guard is never evaluated, which makes the test unable
// to fail.
func TestSet_EmptyContainerTagsAndJSONUnmarshaler(t *testing.T) {
	t.Run("an object tag is withheld", func(t *testing.T) {
		type sample struct {
			Value umJSONRecorder `default:"{}"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Empty(t, got.Value.Raw, "UnmarshalJSON is not called for {}")
	})

	t.Run("an array tag arrives through encoding/json", func(t *testing.T) {
		type sample struct {
			Value umJSONRecorder `default:"[]"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "[]", got.Value.Raw,
			"QUIRK: the interface path withholds [] but the struct path passes it to encoding/json, which calls UnmarshalJSON")
	})

	t.Run("any other value arrives directly", func(t *testing.T) {
		type sample struct {
			Value umJSONRecorder `default:"{\"a\": 1}"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, `{"a": 1}`, got.Value.Raw)
	})

	t.Run("the text unmarshaler is still tried for an object tag", func(t *testing.T) {
		type sample struct {
			Value umBoth `default:"{}"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "text", got.Value.Via, "only the JSON branch withholds {}")
	})
}

// TestSet_TimeTime covers the stdlib struct most likely to carry a default tag. It reaches
// time.Time's UnmarshalText, and a value that does not parse is a hard error rather than a silent
// skip — the one common way a caller meets that path.
func TestSet_TimeTime(t *testing.T) {
	t.Run("RFC 3339", func(t *testing.T) {
		type sample struct {
			At time.Time `default:"2020-01-02T03:04:05Z"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "2020-01-02T03:04:05Z", got.At.Format(time.RFC3339))
	})

	t.Run("unparsable", func(t *testing.T) {
		got := struct {
			At time.Time `default:"nonsense"`
		}{}

		require.Error(t, defaults.Set(&got))
		assert.True(t, got.At.IsZero())
	})
}

// TestSet_UnmarshalerNeedsANonEmptyTag pins that neither interface is consulted for an empty tag, so
// a type that could unmarshal anything gets nothing from one.
//
// The field is still touched, though: net.IP is slice-kinded, so the kind path allocates it empty
// like any other container carrying `default:""`.
func TestSet_UnmarshalerNeedsANonEmptyTag(t *testing.T) {
	type sample struct {
		Both umBoth `default:""`
		IP   net.IP `default:""`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Empty(t, got.Both.Via, "no unmarshaler ran")
	require.NotNil(t, got.IP, "but the container is allocated")
	assert.Empty(t, got.IP)
}
