package defaults_test

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"testing"

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

// TestSet_JSONUnmarshalerSkipsEmptyContainers pins that `{}` and `[]` never reach a custom
// UnmarshalJSON: they mean "allocate an empty one", so the unmarshaler would have nothing to add.
func TestSet_JSONUnmarshalerSkipsEmptyContainers(t *testing.T) {
	type sample struct {
		Object umBoth `default:"{}"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, "text", got.Object.Via, "UnmarshalText is still tried; only the JSON branch skips {}")
}

// TestSet_UnmarshalerNeedsANonEmptyTag pins that neither interface is consulted for an empty tag,
// so a type that could unmarshal anything still gets nothing.
func TestSet_UnmarshalerNeedsANonEmptyTag(t *testing.T) {
	type sample struct {
		Both umBoth `default:""`
		IP   net.IP `default:""`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Empty(t, got.Both.Via)
	assert.Nil(t, got.IP)
}
