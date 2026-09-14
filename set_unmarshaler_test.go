package defaults_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
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

// UmTextAndSetter implements encoding.TextUnmarshaler and defaults.Setter, and appends to Via as
// each one runs, so a test can tell "only text ran" from "both ran" whichever order they take. Its
// name is exported because one test embeds it.
type UmTextAndSetter struct {
	Via string
}

func (u *UmTextAndSetter) UnmarshalText(text []byte) error {
	u.Via += "text:" + string(text)
	return nil
}

func (u *UmTextAndSetter) SetDefaults() {
	u.Via += "+setter"
}

// umJSONListRecorder is slice-kinded and implements only json.Unmarshaler, recording the raw bytes
// it was handed as its one element.
type umJSONListRecorder []string

func (l *umJSONListRecorder) UnmarshalJSON(b []byte) error {
	*l = umJSONListRecorder{string(b)}
	return nil
}

// umJSONMapRecorder is map-kinded and implements only json.Unmarshaler, recording the raw bytes it
// was handed under "raw".
type umJSONMapRecorder map[string]string

func (m *umJSONMapRecorder) UnmarshalJSON(b []byte) error {
	*m = umJSONMapRecorder{"raw": string(b)}
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

// umFailingBoth implements both interfaces and rejects everything, each with an error of its own,
// so a test can tell whose rejection is reported.
type umFailingBoth int

func (u *umFailingBoth) UnmarshalText([]byte) error {
	return errors.New("text fails")
}

func (u *umFailingBoth) UnmarshalJSON([]byte) error {
	return errors.New("json fails")
}

// umDuration wraps a duration to give it a text format, which makes it struct-kinded: parsing by
// kind hands its tag to encoding/json.
type umDuration struct {
	time.Duration
}

func (d *umDuration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// umEndpoint has a text form, "host:port", as well as tags on its fields: the shape of a config
// type that can be given as one string or filled in field by field.
type umEndpoint struct {
	Host string `default:"localhost"`
	Port int    `default:"8080"`
}

func (e *umEndpoint) UnmarshalText(text []byte) error {
	host, port, err := net.SplitHostPort(string(text))
	if err != nil {
		return err
	}
	e.Host = host
	e.Port, err = strconv.Atoi(port)
	return err
}

// umHexID is a fixed-size identifier with a hex text form, the shape of uuid.UUID. It is
// array-kinded, and setField does not parse an array by kind, so a tag UnmarshalText rejects has
// nothing to fall back to.
type umHexID [4]byte

func (id *umHexID) UnmarshalText(text []byte) error {
	if len(text) != hex.EncodedLen(len(id)) {
		return errors.New("invalid ID length")
	}
	_, err := hex.Decode(id[:], text)
	return err
}

// umFailingJSONArray always fails, and is array-kinded, so there is nothing to fall back to.
type umFailingJSONArray [2]int

func (u *umFailingJSONArray) UnmarshalJSON([]byte) error {
	return errors.New("always fails")
}

// umFailingComplex always fails, and is complex-kinded, another kind setField does not parse.
type umFailingComplex complex128

func (u *umFailingComplex) UnmarshalText([]byte) error {
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

// TestSet_TextUnmarshalerArray covers an array type with a text form, the shape of uuid.UUID.
// setField does not parse an array by kind, but the tag is offered to UnmarshalText before the
// kind is looked at, so it fills the field all the same.
func TestSet_TextUnmarshalerArray(t *testing.T) {
	type sample struct {
		ID umHexID `default:"0a1b2c3d"`
	}

	var got sample
	require.NoError(t, defaults.Set(&got))

	assert.Equal(t, umHexID{0x0a, 0x1b, 0x2c, 0x3d}, got.ID)
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

// TestSet_TextUnmarshalerWinsOverSetter pins the precedence the README claims. A successful
// UnmarshalText returns out of setField ahead of the recursion that descends into the struct, and
// SetDefaults is reached only through that recursion -- so the setter never runs for this field.
// The root's own SetDefaults is unaffected, because Set calls it directly; that is
// TestSetter_CalledOnRoot.
//
// A pointer field used to be the exception. The pointer branch of setField called the setter
// itself once the recursion returned, so it ran after UnmarshalText regardless; that call was the
// duplicate in https://github.com/creasty/defaults/issues/67, and removing it brought the pointer
// into line.
//
// An embedded field was another. Go promotes the embedded type's SetDefaults to the struct that
// embeds it, and Set called it through the struct after UnmarshalText had taken the tag.
func TestSet_TextUnmarshalerWinsOverSetter(t *testing.T) {
	t.Run("with a value to unmarshal", func(t *testing.T) {
		got := struct {
			Value UmTextAndSetter `default:"x"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "text:x", got.Value.Via, "UnmarshalText ran and SetDefaults did not")
	})

	t.Run("with an empty tag", func(t *testing.T) {
		got := struct {
			Value UmTextAndSetter `default:""`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "+setter", got.Value.Via, "nothing to unmarshal, so the setter is reached")
	})

	t.Run("behind a pointer", func(t *testing.T) {
		got := struct {
			Value *UmTextAndSetter `default:"x"`
		}{}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Value)
		assert.Equal(t, "text:x", got.Value.Via, "UnmarshalText ran and SetDefaults did not")
	})

	t.Run("embedded", func(t *testing.T) {
		got := struct {
			UmTextAndSetter `default:"x"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, "text:x", got.Via, "UnmarshalText ran and SetDefaults did not")
	})
}

// TestSet_FailingUnmarshalerFallsBackToKind covers what happens when the type's own unmarshaler
// rejects the value: the value is parsed by kind instead, and when that succeeds, the rejection
// goes unreported. When it fails too, the rejection is what Set reports; that is
// TestSet_FailingUnmarshalerErrorIsReported.
//
// The first subtest shows the fall-back with types that reject every value. The others pin tags
// in real use that work only because of it, and that is why it is kept: they all fail if the
// rejection is reported straight away instead, the second option in
// https://github.com/creasty/defaults/issues/79.
func TestSet_FailingUnmarshalerFallsBackToKind(t *testing.T) {
	t.Run("types that reject every value", func(t *testing.T) {
		type sample struct {
			Text umFailingText `default:"hello"`
			JSON umFailingJSON `default:"5"`
		}

		var got sample
		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, umFailingText("hello"), got.Text, "parsed as a string after UnmarshalText failed")
		assert.Equal(t, umFailingJSON(5), got.JSON, "parsed as an int after UnmarshalJSON failed")
	})

	t.Run("an empty object allocates a pointer to a type with a text form", func(t *testing.T) {
		got := struct {
			Endpoint *umEndpoint `default:"{}"`
		}{}

		require.NoError(t, defaults.Set(&got))

		require.NotNil(t, got.Endpoint)
		assert.Equal(t, umEndpoint{Host: "localhost", Port: 8080}, *got.Endpoint,
			"UnmarshalText rejects {}, so the pointer is allocated and its fields get their own tags")
	})

	t.Run("a number sets a slog.Level, whose unmarshalers take names", func(t *testing.T) {
		// Checked first, since a Go release that taught either of them numbers would leave nothing
		// here for the fall-back to do, and this subtest unable to fail.
		require.Error(t, new(slog.Level).UnmarshalText([]byte("4")))
		require.Error(t, new(slog.Level).UnmarshalJSON([]byte("4")))

		got := struct {
			Level slog.Level `default:"4"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, slog.LevelWarn, got.Level, "parsed as an int after both unmarshalers rejected it")
	})
}

// TestSet_FailingUnmarshalerErrorIsReported covers a tag the type's own unmarshaler rejects and
// parsing by kind cannot take either. The error Set returns names that rejection as its cause.
// For a type that implements both interfaces, the tag goes to UnmarshalText first, so its
// rejection is the one named.
//
// The cause used to be the failed parse by kind, because the rejection was discarded. For a
// struct-kinded type, that was a syntax error from encoding/json, a parser the tag was never
// written for. See https://github.com/creasty/defaults/issues/79.
func TestSet_FailingUnmarshalerErrorIsReported(t *testing.T) {
	type textOnly struct {
		Timeout umDuration `default:"garbage"`
	}
	type jsonOnly struct {
		Level umFailingJSON `default:"x"`
	}
	type both struct {
		Level umFailingBoth `default:"x"`
	}

	tests := []struct {
		name string
		ptr  interface{}
		want string
	}{
		{"text", &textOnly{}, `field Timeout: invalid default "garbage": time: invalid duration "garbage"`},
		{"JSON", &jsonOnly{}, `field Level: invalid default "x": always fails`},
		{"both", &both{}, `field Level: invalid default "x": text fails`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, defaults.Set(tt.ptr), tt.want)
		})
	}
}

// TestSet_FailingUnmarshalerWithNothingToFallBackTo covers a tag the type's own unmarshaler
// rejects when setField does not parse the type's kind at all: an array, as uuid.UUID is, or a
// complex number. There is no parse to fail, but the rejection is reported just as it is when one
// fails; that is TestSet_FailingUnmarshalerErrorIsReported. A type with no unmarshaler has no
// rejection to report, so its tag is still ignored; that is TestSet_ArraysAreLeftAlone.
//
// The rejection used to be dropped. The field kept its zero value and Set returned nil, leaving a
// value that looks legitimate for the types this hits: the nil UUID, or an all-zero hash. See
// https://github.com/creasty/defaults/issues/89.
func TestSet_FailingUnmarshalerWithNothingToFallBackTo(t *testing.T) {
	type array struct {
		ID umHexID `default:"not-an-id"`
	}
	type pointer struct {
		ID *umHexID `default:"not-an-id"`
	}
	type jsonOnly struct {
		Pair umFailingJSONArray `default:"x"`
	}
	type complexNumber struct {
		Value umFailingComplex `default:"x"`
	}

	tests := []struct {
		name string
		ptr  interface{}
		want string
	}{
		{"array", &array{}, `field ID: invalid default "not-an-id": invalid ID length`},
		{"pointer", &pointer{}, `field ID: invalid default "not-an-id": invalid ID length`},
		{"JSON", &jsonOnly{}, `field Pair: invalid default "x": always fails`},
		{"complex", &complexNumber{}, `field Value: invalid default "x": always fails`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, defaults.Set(tt.ptr), tt.want)
		})
	}
}

// TestSet_EmptyContainerTagsAndJSONUnmarshaler pins which empty-container tags reach a custom
// UnmarshalJSON. The interface path withholds both `{}` and `[]` whatever the field's kind, on the
// grounds that they mean "allocate an empty one" — but the kind-based path allocates an empty value
// only for its own literal, `{}` for a struct or map and `[]` for a slice, and hands the other one to
// encoding/json, which calls the very same method. So the two are not symmetric. A scalar kind gets
// neither: it parses the literal itself, which a number or a bool rejects and a string takes as it
// is.
//
// This needs types that implement *only* json.Unmarshaler. With one that also implements
// UnmarshalText, text wins first and the JSON guard is never evaluated, which makes the test unable
// to fail.
func TestSet_EmptyContainerTagsAndJSONUnmarshaler(t *testing.T) {
	t.Run("a slice or map type's own literal is withheld", func(t *testing.T) {
		got := struct {
			List umJSONListRecorder `default:"[]"`
			Map  umJSONMapRecorder  `default:"{}"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, umJSONListRecorder{}, got.List, "allocated empty, and UnmarshalJSON is not called")
		assert.Equal(t, umJSONMapRecorder{}, got.Map, "allocated empty, and UnmarshalJSON is not called")
	})

	t.Run("a slice or map type's other literal arrives through encoding/json", func(t *testing.T) {
		got := struct {
			List umJSONListRecorder `default:"{}"`
			Map  umJSONMapRecorder  `default:"[]"`
		}{}

		require.NoError(t, defaults.Set(&got))

		assert.Equal(t, umJSONListRecorder{"{}"}, got.List,
			"QUIRK: the interface path withholds {} but the slice path passes it to encoding/json, which calls UnmarshalJSON")
		assert.Equal(t, umJSONMapRecorder{"raw": "[]"}, got.Map,
			"QUIRK: the interface path withholds [] but the map path passes it to encoding/json, which calls UnmarshalJSON")
	})

	t.Run("a scalar type parses the literal itself", func(t *testing.T) {
		got := struct {
			Value umJSONEnum `default:"[]"`
		}{}

		assert.EqualError(t, defaults.Set(&got), `field Value: invalid default "[]": strconv.ParseInt: parsing "[]": invalid syntax`,
			"QUIRK: UnmarshalJSON is never offered the tag, so the error is the int parse's rather than its rejection")
	})

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

		err := defaults.Set(&got)

		require.Error(t, err)
		var parseErr *time.ParseError
		assert.ErrorAs(t, err, &parseErr, "the cause is UnmarshalText's rejection, not the JSON fall-back's")
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
