package defaults

import (
	"reflect"
)

// Setter is an interface for setting default values
type Setter interface {
	SetDefaults()
}

func callSetter(v interface{}) {
	if ds, ok := v.(Setter); ok {
		ds.SetDefaults()
	}
}

// CanUpdate returns true when the given value is an initial value of its type
func CanUpdate(v interface{}) bool {
	val := reflect.ValueOf(v)
	return reflect.DeepEqual(reflect.Zero(val.Type()).Interface(), val.Interface())
}
