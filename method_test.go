package defaults

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This test is in package defaults rather than defaults_test: isPromotedMethod is unexported, and Set
// reaches it only with a struct, a setter and a walk, to ask one question about a type. What Set makes
// of the answer is pinned from outside, in setter_test.go.
//
// The fixtures declare a method of no particular name, since the file knows of none: which method the
// walk cares about is setter.go's business. The types are package-level because each carries a method,
// and named with a file-unique prefix.

// methodValue declares Ping on a value receiver.
type methodValue struct{}

func (methodValue) Ping() {}

// methodPointer declares it on a pointer receiver, so only a pointer to it has the method.
type methodPointer struct{}

func (*methodPointer) Ping() {}

// methodEmbedsValue takes it from a field, promoted.
type methodEmbedsValue struct{ methodValue }

// methodEmbedsPointerReceiver embeds a value whose method has a pointer receiver, so the method is in
// the embedding pointer's set alone.
type methodEmbedsPointerReceiver struct{ methodPointer }

// methodEmbedsPointerField embeds a pointer, and takes the method through it.
type methodEmbedsPointerField struct{ *methodValue }

// methodEmbedsTwice takes it from a field that took it from a field.
type methodEmbedsTwice struct{ methodEmbedsValue }

// methodShadows declares its own over the one it would take from its field, and calls through to it,
// as a type that means to add to an embedded method does.
type methodShadows struct{ methodValue }

func (s methodShadows) Ping() { s.methodValue.Ping() }

// methodPinger is embedded as an interface, so the method is promoted from a field that holds it.
type methodPinger interface{ Ping() }

type methodEmbedsInterface struct{ methodPinger }

// TestIsPromotedMethod covers where the code behind a method comes from, which reflect reports alike
// for a method a type declares and one it takes from an embedded field.
func TestIsPromotedMethod(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{"declared on a value receiver", reflect.TypeOf(methodValue{}), false},
		{"declared on a pointer receiver", reflect.TypeOf(methodPointer{}), false},
		{"declared over an embedded field's", reflect.TypeOf(methodShadows{}), false},
		{"taken from an embedded value", reflect.TypeOf(methodEmbedsValue{}), true},
		{"taken from an embedded value, pointer receiver", reflect.TypeOf(methodEmbedsPointerReceiver{}), true},
		{"taken from an embedded pointer", reflect.TypeOf(methodEmbedsPointerField{}), true},
		{"taken from a field that took it", reflect.TypeOf(methodEmbedsTwice{}), true},
		{"taken from an embedded interface", reflect.TypeOf(methodEmbedsInterface{}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPromotedMethod(tt.typ, "Ping"))
		})
	}
}
