package defaults

import (
	"errors"
	"reflect"
	"strconv"
	"time"
)

var (
	errInvalidType = errors.New("not a struct pointer")
)

const defaultFieldName = "default"

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
		if defaultVal := t.Field(i).Tag.Get(defaultFieldName); defaultVal != "" {
			setField(v.Field(i), defaultVal)
		}
	}

	return nil
}

func setField(field reflect.Value, defaultVal string) {
	var val interface{}
	var err error

	switch field.Kind() {
	case reflect.Bool:
		val, err = strconv.ParseBool(defaultVal)
	case reflect.Int:
		val, err = strconv.ParseInt(defaultVal, 10, 64)
		val = int(val.(int64))
	case reflect.Int8:
		val, err = strconv.ParseInt(defaultVal, 10, 8)
		val = int8(val.(int64))
	case reflect.Int16:
		val, err = strconv.ParseInt(defaultVal, 10, 16)
		val = int16(val.(int64))
	case reflect.Int32:
		val, err = strconv.ParseInt(defaultVal, 10, 32)
		val = int32(val.(int64))
	case reflect.Int64:
		if t, err := time.ParseDuration(defaultVal); err == nil {
			val = t
		} else {
			val, err = strconv.ParseInt(defaultVal, 10, 64)
		}
	case reflect.Uint:
		val, err = strconv.ParseUint(defaultVal, 10, 64)
		val = uint(val.(uint64))
	case reflect.Uint8:
		val, err = strconv.ParseUint(defaultVal, 10, 8)
		val = uint8(val.(uint64))
	case reflect.Uint16:
		val, err = strconv.ParseUint(defaultVal, 10, 16)
		val = uint16(val.(uint64))
	case reflect.Uint32:
		val, err = strconv.ParseUint(defaultVal, 10, 32)
		val = uint32(val.(uint64))
	case reflect.Uint64:
		val, err = strconv.ParseUint(defaultVal, 10, 64)
	case reflect.Uintptr:
		val, err = strconv.ParseUint(defaultVal, 10, 64)
		val = uintptr(val.(uint64))
	case reflect.Float32:
		val, err = strconv.ParseFloat(defaultVal, 32)
		val = float32(val.(float64))
	case reflect.Float64:
		val, err = strconv.ParseFloat(defaultVal, 64)
	case reflect.String:
		val = defaultVal
	default:
		return
	}

	if err != nil {
		return
	}
	if field.CanSet() {
		field.Set(reflect.ValueOf(val))
	}
}
