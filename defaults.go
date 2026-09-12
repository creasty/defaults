package defaults

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	errInvalidType = errors.New("not a struct pointer")
)

const (
	fieldName = "default"
)

// fieldTag is the `default` tag as it applies to one field: the field's name, for error messages,
// the tag's value, and whether the tag was present at all — an empty value and an absent tag mean
// different things. Grouping them keeps setField to one context parameter rather than three
// positional ones, two of them strings.
type fieldTag struct {
	fieldName string
	value     string
	present   bool
}

// Set initializes members in a struct referenced by a pointer.
// Maps and slices are initialized by `make` and other primitive types are set with default values.
// `ptr` should be a struct pointer
func Set(ptr interface{}) error {
	if reflect.TypeOf(ptr).Kind() != reflect.Pointer {
		return errInvalidType
	}

	v := reflect.ValueOf(ptr).Elem()
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return errInvalidType
	}

	for i := 0; i < t.NumField(); i++ {
		defaultVal, ok := t.Field(i).Tag.Lookup(fieldName)
		if ok && defaultVal == "-" {
			continue
		}

		if err := setField(v.Field(i), fieldTag{
			fieldName: t.Field(i).Name,
			value:     defaultVal,
			present:   ok,
		}); err != nil {
			return err
		}
	}
	callSetter(ptr)
	return nil
}

// MustSet function is a wrapper of Set function
// It will call Set and panic if err not equals nil.
func MustSet(ptr interface{}) {
	if err := Set(ptr); err != nil {
		panic(err)
	}
}

func setField(field reflect.Value, tag fieldTag) error {
	// Kept as a local because the parsing below reads better against a plain name.
	defaultVal := tag.value

	// parseErr turns a failed parse into the error Set returns, with one exception: an empty tag asks
	// for this type's zero value rather than for anything to be parsed, so there is nothing to report
	// and the field keeps the value it already has.
	parseErr := func(err error) error {
		if defaultVal == "" {
			return nil
		}

		return fmt.Errorf("field %s: invalid default %q: %w", tag.fieldName, defaultVal, err)
	}

	if !field.CanSet() {
		return nil
	}

	if !tag.present && !shouldInitializeField(field) {
		return nil
	}

	isInitial := isInitialValue(field)
	if isInitial {
		if unmarshalByInterface(field, defaultVal) {
			return nil
		}

		switch field.Kind() {
		case reflect.Bool:
			val, err := strconv.ParseBool(defaultVal)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(val).Convert(field.Type()))
		case reflect.Int:
			val, err := strconv.ParseInt(defaultVal, 0, strconv.IntSize)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(int(val)).Convert(field.Type()))
		case reflect.Int8:
			val, err := strconv.ParseInt(defaultVal, 0, 8)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(int8(val)).Convert(field.Type()))
		case reflect.Int16:
			val, err := strconv.ParseInt(defaultVal, 0, 16)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(int16(val)).Convert(field.Type()))
		case reflect.Int32:
			val, err := strconv.ParseInt(defaultVal, 0, 32)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(int32(val)).Convert(field.Type()))
		case reflect.Int64:
			// The duration attempt tolerates surrounding whitespace. Doing it here rather than
			// against time.Duration's exact type covers named duration types too, and leaves the
			// numeric fallback strict.
			if val, err := time.ParseDuration(strings.TrimSpace(defaultVal)); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			} else if val, err := strconv.ParseInt(defaultVal, 0, 64); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			} else {
				return parseErr(err)
			}
		case reflect.Uint:
			val, err := strconv.ParseUint(defaultVal, 0, strconv.IntSize)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(uint(val)).Convert(field.Type()))
		case reflect.Uint8:
			val, err := strconv.ParseUint(defaultVal, 0, 8)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(uint8(val)).Convert(field.Type()))
		case reflect.Uint16:
			val, err := strconv.ParseUint(defaultVal, 0, 16)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(uint16(val)).Convert(field.Type()))
		case reflect.Uint32:
			val, err := strconv.ParseUint(defaultVal, 0, 32)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(uint32(val)).Convert(field.Type()))
		case reflect.Uint64:
			val, err := strconv.ParseUint(defaultVal, 0, 64)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(val).Convert(field.Type()))
		case reflect.Uintptr:
			val, err := strconv.ParseUint(defaultVal, 0, strconv.IntSize)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(uintptr(val)).Convert(field.Type()))
		case reflect.Float32:
			val, err := strconv.ParseFloat(defaultVal, 32)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(float32(val)).Convert(field.Type()))
		case reflect.Float64:
			val, err := strconv.ParseFloat(defaultVal, 64)
			if err != nil {
				return parseErr(err)
			}
			field.Set(reflect.ValueOf(val).Convert(field.Type()))
		case reflect.String:
			field.Set(reflect.ValueOf(defaultVal).Convert(field.Type()))

		case reflect.Slice:
			ref := reflect.New(field.Type())
			ref.Elem().Set(reflect.MakeSlice(field.Type(), 0, 0))
			if defaultVal != "" && defaultVal != "[]" {
				if err := json.Unmarshal([]byte(defaultVal), ref.Interface()); err != nil {
					return parseErr(err)
				}
			}
			field.Set(ref.Elem().Convert(field.Type()))
		case reflect.Map:
			ref := reflect.New(field.Type())
			ref.Elem().Set(reflect.MakeMap(field.Type()))
			if defaultVal != "" && defaultVal != "{}" {
				if err := json.Unmarshal([]byte(defaultVal), ref.Interface()); err != nil {
					return parseErr(err)
				}
			}
			field.Set(ref.Elem().Convert(field.Type()))
		case reflect.Struct:
			if defaultVal != "" && defaultVal != "{}" {
				if err := json.Unmarshal([]byte(defaultVal), field.Addr().Interface()); err != nil {
					return parseErr(err)
				}
			}
		case reflect.Pointer:
			field.Set(reflect.New(field.Type().Elem()))
		}
	}

	switch field.Kind() {
	case reflect.Pointer:
		if isInitial || field.Elem().Kind() == reflect.Struct {
			if err := setField(field.Elem(), tag); err != nil {
				return err
			}
			callSetter(field.Interface())
		}
	case reflect.Struct:
		if err := Set(field.Addr().Interface()); err != nil {
			return err
		}
	case reflect.Slice:
		for j := 0; j < field.Len(); j++ {
			if err := setField(field.Index(j), fieldTag{fieldName: tag.fieldName}); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, e := range field.MapKeys() {
			v := field.MapIndex(e)

			// A pointer value is written through, so it needs no copy and no write-back. Everything
			// else does: a map value is not addressable, so it is worked on as a copy and put back.
			originalIsPtr := v.Kind() == reflect.Pointer
			if originalIsPtr {
				if v.IsNil() {
					continue
				}
				v = v.Elem()
			}

			switch v.Kind() {
			case reflect.Struct, reflect.Slice, reflect.Map:
				if !v.CanAddr() {
					copyValue := reflect.New(v.Type())
					copyValue.Elem().Set(v)
					v = copyValue.Elem()
				}

				if err := setField(v, fieldTag{fieldName: tag.fieldName}); err != nil {
					return err
				}

				if !originalIsPtr {
					field.SetMapIndex(e, v)
				}
			}
		}
	}

	return nil
}

func unmarshalByInterface(field reflect.Value, defaultVal string) bool {
	asText, ok := field.Addr().Interface().(encoding.TextUnmarshaler)
	if ok && defaultVal != "" {
		// if field implements encode.TextUnmarshaler, try to use it before decode by kind
		if err := asText.UnmarshalText([]byte(defaultVal)); err == nil {
			return true
		}
	}
	asJSON, ok := field.Addr().Interface().(json.Unmarshaler)
	if ok && defaultVal != "" && defaultVal != "{}" && defaultVal != "[]" {
		// if field implements json.Unmarshaler, try to use it before decode by kind
		if err := asJSON.UnmarshalJSON([]byte(defaultVal)); err == nil {
			return true
		}
	}
	return false
}

func isInitialValue(field reflect.Value) bool {
	if !field.IsValid() {
		return true
	}
	return reflect.DeepEqual(reflect.Zero(field.Type()).Interface(), field.Interface())
}

// shouldInitializeField reports whether the field's own state warrants visiting it, regardless of
// any tag: a struct is always descended into, as is a pointer the caller already allocated, and a
// container the caller already filled has elements to recurse into. Whether a tag is present is the
// caller's business.
func shouldInitializeField(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.Struct:
		return true
	case reflect.Pointer:
		return !field.IsNil() && field.Elem().Kind() == reflect.Struct
	case reflect.Slice, reflect.Map:
		return field.Len() > 0
	}

	return false
}

// CanUpdate returns true when the given value is an initial value of its type
func CanUpdate(v interface{}) bool {
	return isInitialValue(reflect.ValueOf(v))
}
