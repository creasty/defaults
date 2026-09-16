package defaults

import (
	"reflect"
)

// Setter is an interface for setting default values.
//
// Set calls SetDefaults on a struct once it has filled the struct's fields, and again for every other
// path that reaches the same struct, though not for a cycle leading back to a struct still being
// filled. A SetDefaults the struct has only by promotion from an embedded field is not called on the
// struct: it belongs to the field, which gets whatever call it would get as a named field.
type Setter interface {
	SetDefaults()
}

func callSetter(v interface{}) {
	if ds, ok := v.(Setter); ok {
		ds.SetDefaults()
	}
}

// hasPromotedSetter reports whether the SetDefaults of t, a struct type whose pointer implements
// Setter, is promoted from an embedded field rather than declared by t itself. How that is told
// apart is isPromotedMethod's business; which method is Setter's, so the name stays here.
func hasPromotedSetter(t reflect.Type) bool {
	return isPromotedMethod(t, "SetDefaults")
}
