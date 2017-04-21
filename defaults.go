package defaults

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"
)

var (
	errInvalidType = errors.New("not a struct pointer")
)

const (
	fieldName = "default"
)

// Initializer is an interface for setting default values
type Initializer interface {
	SetDefaults()
}

// Init initializes members in a struct referenced by a pointer.
// Maps and slices are initialized by `make` and other primitive types are set with default values.
// `ptr` should be a struct pointer
func Init(ptr interface{}) error {
	if reflect.TypeOf(ptr).Kind() != reflect.Ptr {
		return errInvalidType
	}

	v := reflect.ValueOf(ptr).Elem()
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return errInvalidType
	}

	for i := 0; i < t.NumField(); i++ {
		if defaultVal := t.Field(i).Tag.Get(fieldName); defaultVal != "" {
			setField(v.Field(i), defaultVal)
		}
	}

	return nil
}

func setField(field reflect.Value, defaultVal string) {
	if !field.CanSet() {
		return
	}

	switch field.Kind() {
	case reflect.Bool:
		if val, err := strconv.ParseBool(defaultVal); err == nil {
			field.Set(reflect.ValueOf(val))
		}
	case reflect.Int:
		if val, err := strconv.ParseInt(defaultVal, 10, 64); err == nil {
			field.Set(reflect.ValueOf(int(val)))
		}
	case reflect.Int8:
		if val, err := strconv.ParseInt(defaultVal, 10, 8); err == nil {
			field.Set(reflect.ValueOf(int8(val)))
		}
	case reflect.Int16:
		if val, err := strconv.ParseInt(defaultVal, 10, 16); err == nil {
			field.Set(reflect.ValueOf(int16(val)))
		}
	case reflect.Int32:
		if val, err := strconv.ParseInt(defaultVal, 10, 32); err == nil {
			field.Set(reflect.ValueOf(int32(val)))
		}
	case reflect.Int64:
		if t, err := time.ParseDuration(defaultVal); err == nil {
			field.Set(reflect.ValueOf(t))
		} else if val, err := strconv.ParseInt(defaultVal, 10, 64); err == nil {
			field.Set(reflect.ValueOf(val))
		}
	case reflect.Uint:
		if val, err := strconv.ParseUint(defaultVal, 10, 64); err == nil {
			field.Set(reflect.ValueOf(uint(val)))
		}
	case reflect.Uint8:
		if val, err := strconv.ParseUint(defaultVal, 10, 8); err == nil {
			field.Set(reflect.ValueOf(uint8(val)))
		}
	case reflect.Uint16:
		if val, err := strconv.ParseUint(defaultVal, 10, 16); err == nil {
			field.Set(reflect.ValueOf(uint16(val)))
		}
	case reflect.Uint32:
		if val, err := strconv.ParseUint(defaultVal, 10, 32); err == nil {
			field.Set(reflect.ValueOf(uint32(val)))
		}
	case reflect.Uint64:
		if val, err := strconv.ParseUint(defaultVal, 10, 64); err == nil {
			field.Set(reflect.ValueOf(val))
		}
	case reflect.Uintptr:
		if val, err := strconv.ParseUint(defaultVal, 10, 64); err == nil {
			field.Set(reflect.ValueOf(uintptr(val)))
		}
	case reflect.Float32:
		if val, err := strconv.ParseFloat(defaultVal, 32); err == nil {
			field.Set(reflect.ValueOf(float32(val)))
		}
	case reflect.Float64:
		if val, err := strconv.ParseFloat(defaultVal, 64); err == nil {
			field.Set(reflect.ValueOf(val))
		}

	case reflect.String:
		field.Set(reflect.ValueOf(defaultVal))
	case reflect.Slice:
		val := reflect.New(field.Type())
		val.Elem().Set(reflect.MakeSlice(field.Type(), 0, 0))
		json.Unmarshal([]byte(defaultVal), val.Interface())
		field.Set(val.Elem())
	case reflect.Map:
		val := reflect.New(field.Type())
		val.Elem().Set(reflect.MakeMap(field.Type()))
		json.Unmarshal([]byte(defaultVal), val.Interface())
		field.Set(val.Elem())
	case reflect.Struct:
		val := reflect.New(field.Type())
		json.Unmarshal([]byte(defaultVal), val.Interface())
		Init(val.Interface())
		field.Set(val.Elem())
	case reflect.Ptr:
		val := reflect.New(field.Type().Elem())
		field.Set(val)
		setField(val.Elem(), defaultVal)
	}

	if initializer, ok := field.Interface().(Initializer); ok {
		initializer.SetDefaults()
	}
}
