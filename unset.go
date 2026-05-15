package defaults

import (
	"reflect"
)

const (
	unsetFlag = "unset"
	// unsetRecursion means that the field will be unset recursively
	unsetRecursion = "walk"
)

func MustUnset(ptr interface{}) {
	if err := Unset(ptr); err != nil {
		panic(err)
	}
}
func Unset(obj interface{}) error {
	v := indirect(reflect.ValueOf(obj))
	if v.Kind() != reflect.Struct {
		return errInvalidType
	}
	unsetStruct(v)
	return nil
}
func unsetStruct(obj reflect.Value) {
	t := obj.Type()
	for i := 0; i < t.NumField(); i++ {
		unsetVal := t.Field(i).Tag.Get(unsetFlag)
		if unsetVal == "-" {
			continue
		}
		unsetField(obj.Field(i), unsetVal == unsetRecursion)
	}
}

func indirect(v reflect.Value) reflect.Value {
	finalValue := v
	for finalValue.Kind() == reflect.Ptr {
		finalValue = finalValue.Elem()
	}
	return finalValue
}

func unsetField(field reflect.Value, unsetWalk bool) {
	if !field.CanSet() {
		return
	}

	isInitial := isInitialValue(field)
	if isInitial {
		return
	}
	if !unsetWalk {
		field.Set(reflect.Zero(field.Type()))
		return
	}

	switch field.Kind() {
	default:
		field.Set(reflect.Zero(field.Type()))
	case reflect.Ptr:
		unsetField(field.Elem(), true)
	case reflect.Struct:
		unsetStruct(field)
	case reflect.Slice:
		for j := 0; j < field.Len(); j++ {
			unsetField(field.Index(j), true)
		}
	case reflect.Map:
		for _, e := range field.MapKeys() {
			var mapValue = field.MapIndex(e)
			switch mapValue.Kind() {
			case reflect.Ptr:
				unsetField(mapValue.Elem(), true)
			case reflect.Struct, reflect.Slice, reflect.Map:
				ref := reflect.New(mapValue.Type())
				ref.Elem().Set(mapValue)
				unsetField(ref.Elem(), true)
				field.SetMapIndex(e, ref.Elem().Convert(mapValue.Type()))
			default:
				field.SetMapIndex(e, reflect.Zero(mapValue.Type()))
			}
		}
	}
}
