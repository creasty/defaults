package defaults

import "reflect"

func isInitialValue(field reflect.Value) bool {
	if !field.IsValid() {
		return true
	}
	return reflect.DeepEqual(reflect.Zero(field.Type()).Interface(), field.Interface())
}

// CanUpdate returns true when the given value is an initial value of its type
func CanUpdate(v interface{}) bool {
	return isInitialValue(reflect.ValueOf(v))
}
