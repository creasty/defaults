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

// ErrInvalidType is the error Set returns, and MustSet panics with, when the argument is not a
// non-nil pointer to a struct. Test for it with errors.Is.
var ErrInvalidType = errors.New("not a struct pointer")

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

// pendingDefault is a tag being applied to a zero value further up the current path, linked to the
// one being applied above it.
type pendingDefault struct {
	typ   reflect.Type
	value string
	outer *pendingDefault
}

// Set initializes members in a struct referenced by a pointer.
// Maps and slices are initialized by `make` and other primitive types are set with default values.
// `ptr` should be a non-nil struct pointer, or Set returns ErrInvalidType.
func Set(ptr interface{}) error {
	return set(ptr, nil)
}

// set is Set, for a struct that may be reached by recursion and so carries the tags still being
// applied above it.
func set(ptr interface{}, pending *pendingDefault) error {
	// The kind is read off the Value because reflect.TypeOf(nil) is itself nil, and a nil pointer
	// has no struct behind it to fill: its Elem is an invalid Value, which has no Type. See
	// https://github.com/creasty/defaults/issues/69.
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return ErrInvalidType
	}

	v = v.Elem()
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return ErrInvalidType
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
		}, pending); err != nil {
			return err
		}
	}

	// A SetDefaults promoted from an embedded field is that field's, and the loop above has dealt with
	// it there, just as for a named field. Calling it through the struct as well would run it again.
	if s, ok := ptr.(Setter); ok && !hasPromotedSetter(t) {
		s.SetDefaults()
	}
	return nil
}

func setField(field reflect.Value, tag fieldTag, pending *pendingDefault) error {
	// Kept as a local because the parsing below reads better against a plain name.
	defaultVal := tag.value

	if !field.CanSet() {
		return nil
	}

	if !tag.present && !shouldInitializeField(field) {
		return nil
	}

	isInitial := isInitialValue(field)
	if isInitial {
		// What a tag makes of a zero value depends on nothing but the value's type and the tag. So
		// meeting the same pair below where it is already being applied means meeting it again below
		// that, without end: a default that creates another of itself, which used to recurse until
		// the stack overflowed. A value that is not zero never counts, since what happens below it
		// depends on what it holds, and that is how a recursive type ends. See
		// https://github.com/creasty/defaults/issues/71.
		if tag.present {
			for p := pending; p != nil; p = p.outer {
				if p.typ == field.Type() && p.value == defaultVal {
					return fmt.Errorf("field %s: default %q recurses without end", tag.fieldName, defaultVal)
				}
			}
			pending = &pendingDefault{typ: field.Type(), value: defaultVal, outer: pending}
		}

		unmarshaled, unmarshalErr := unmarshalByInterface(field, defaultVal)
		if unmarshaled {
			return nil
		}

		// parseErr turns a failed parse into the error Set returns, with one exception: an empty tag
		// asks for this type's zero value rather than for anything to be parsed, so there is nothing to
		// report and the field keeps the value it already has.
		//
		// When the type's own unmarshaler rejected the tag, its error is the cause reported. Parsing by
		// kind was only the fall-back, and its failure names a parser the tag was never written for:
		// encoding/json, for a struct. See https://github.com/creasty/defaults/issues/79.
		parseErr := func(err error) error {
			if defaultVal == "" {
				return nil
			}
			if unmarshalErr != nil {
				err = unmarshalErr
			}

			return fmt.Errorf("field %s: invalid default %q: %w", tag.fieldName, defaultVal, err)
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
			// Every int64-kinded field is offered to time.ParseDuration, because reflection cannot
			// tell a named duration type from any other int64: matching time.Duration's exact type
			// would leave those types behind. So a plain int64 takes "1h" too, and a bare "1" on a
			// Duration is 1ns, as the README documents. See
			// https://github.com/creasty/defaults/issues/66.
			//
			// The duration attempt tolerates surrounding whitespace, the numeric fallback does not.
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
		default:
			// The kinds left, an array or a complex number among them, are not parsed at all, so a
			// tag the type's own unmarshaler rejected has nothing to fall back to. The rejection is
			// reported rather than dropped, which would leave a zero value that looks legitimate:
			// the nil UUID, for uuid.UUID. See https://github.com/creasty/defaults/issues/89.
			if unmarshalErr != nil {
				return parseErr(unmarshalErr)
			}
		}
	}

	switch field.Kind() {
	case reflect.Pointer:
		if isInitial || field.Elem().Kind() == reflect.Struct {
			if err := setField(field.Elem(), tag, pending); err != nil {
				return err
			}

			// A struct pointee's setter is not called here: Set has called it as the recursion
			// finished, or skipped it because it is promoted or because an unmarshaler took the tag.
			// Anything else behind a pointer gets no setter call but this one.
			if field.Elem().Kind() != reflect.Struct {
				callSetter(field.Interface())
			}
		}
	case reflect.Struct:
		if err := set(field.Addr().Interface(), pending); err != nil {
			return err
		}
	case reflect.Slice:
		for j := 0; j < field.Len(); j++ {
			if err := setField(field.Index(j), fieldTag{fieldName: tag.fieldName}, pending); err != nil {
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

				if err := setField(v, fieldTag{fieldName: tag.fieldName}, pending); err != nil {
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

// unmarshalByInterface offers the tag to the field's own unmarshalers ahead of parsing by kind,
// and reports whether one took it. When none did, the error is the rejection that setField
// reports unless parsing by kind takes the tag instead, or nil when neither unmarshaler was
// offered the tag.
func unmarshalByInterface(field reflect.Value, defaultVal string) (bool, error) {
	var textErr, jsonErr error

	asText, ok := field.Addr().Interface().(encoding.TextUnmarshaler)
	if ok && defaultVal != "" {
		// if field implements encode.TextUnmarshaler, try to use it before decode by kind
		if textErr = asText.UnmarshalText([]byte(defaultVal)); textErr == nil {
			return true, nil
		}
	}
	asJSON, ok := field.Addr().Interface().(json.Unmarshaler)
	if ok && defaultVal != "" && defaultVal != "{}" && defaultVal != "[]" {
		// if field implements json.Unmarshaler, try to use it before decode by kind
		if jsonErr = asJSON.UnmarshalJSON([]byte(defaultVal)); jsonErr == nil {
			return true, nil
		}
	}

	// UnmarshalText is offered the tag first, so if both rejected it, its rejection is reported.
	if textErr != nil {
		return false, textErr
	}
	return false, jsonErr
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
