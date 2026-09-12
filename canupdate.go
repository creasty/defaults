package defaults

import "reflect"

func isInitialValue(field reflect.Value) bool {
	// An invalid Value carries no type, so IsZero would panic on it. There is nothing there to
	// preserve either: see https://github.com/creasty/defaults/issues/47.
	if !field.IsValid() {
		return true
	}
	return field.IsZero()
}

// CanUpdate returns true when the given value is an initial value of its type
func CanUpdate(v interface{}) bool {
	return isInitialValue(reflect.ValueOf(v))
}
