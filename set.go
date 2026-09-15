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
	"unsafe"
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

// walking is a value being walked on the current path, linked to the one above it: a struct, by its
// address, or a slice or map, by the array or table it refers to. Data the caller built with a
// way back to itself leads the walk back to such a value, which Set used to follow until the stack
// overflowed. The walk goes no further there instead, and the value is finished by the walk already
// under way above. Nothing is kept once a value's walk returns, so a value that two paths reach
// without either passing through the other is walked on both.
//
// The type is part of the name, since a struct shares its address with its first field and an array
// with its first element, and so is a slice's length, since a longer slice over the same array has
// elements the shorter one lacks. The address is held as an unsafe.Pointer, which keeps the value
// alive while it is on the path, so no value made below can be given the same address.
//
// Each check scans the path above, so a walk n values deep makes on the order of n²/2 comparisons.
type walking struct {
	ptr   unsafe.Pointer
	typ   reflect.Type
	len   int
	outer *walking
}

// repeats reports whether the value w names is already being walked above it, on w.outer's path.
func (w *walking) repeats() bool {
	for p := w.outer; p != nil; p = p.outer {
		if p.ptr == w.ptr && p.typ == w.typ && p.len == w.len {
			return true
		}
	}
	return false
}

// Set fills the fields of the struct ptr points to from their `default` tags, and descends into
// structs, pointers, slices and maps to do the same below. A field is written only while it holds
// its zero value, and a nil map, slice or pointer is allocated only by a tag.
//
// A value reachable by more than one path is filled on each. Where data with a cycle leads back to a
// value still being walked, Set goes no further, so a cycle ends.
//
// Set stops at the first field whose default fails and returns its error. A value Set found zero on
// the way to that default is left zero again, so a second Set fails again; fields filled elsewhere
// before the failure keep their defaults.
//
// ptr should be a non-nil struct pointer, or Set returns ErrInvalidType.
func Set(ptr interface{}) error {
	return set(ptr, nil, nil)
}

// set is Set, for a struct that may be reached by recursion and so carries the tags still being
// applied above it and the values being walked on the path to it.
func set(ptr interface{}, pending *pendingDefault, path *walking) error {
	// The kind is read off the Value because reflect.TypeOf(nil) is itself nil, and a nil pointer
	// has no struct behind it to fill: its Elem is an invalid Value, which has no Type. See
	// https://github.com/creasty/defaults/issues/69.
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return ErrInvalidType
	}

	p := v.UnsafePointer()
	v = v.Elem()
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return ErrInvalidType
	}

	// The struct type, not ptr's: a caller may hand Set a named pointer type, while the recursion
	// below reaches the same struct as a plain *T.
	here := walking{ptr: p, typ: t, outer: path}
	if here.repeats() {
		return nil
	}

	for i := 0; i < t.NumField(); i++ {
		defaultVal, ok := t.Field(i).Tag.Lookup(fieldName)
		if ok && defaultVal == "-" {
			continue
		}

		if _, err := setField(v.Field(i), fieldTag{
			fieldName: t.Field(i).Name,
			value:     defaultVal,
			present:   ok,
		}, pending, &here); err != nil {
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

// setField fills field from its tag and descends into it. It reports whether an UnmarshalText or
// UnmarshalJSON took the tag, the field's own or, behind a pointer, its pointee's, which decides
// whether the pointer above calls a setter.
func setField(field reflect.Value, tag fieldTag, pending *pendingDefault, path *walking) (bool, error) {
	if !field.CanSet() {
		return false, nil
	}

	if !tag.present && !shouldInitializeField(field) {
		return false, nil
	}

	isInitial := isInitialValue(field)
	taken, err := fillField(field, tag, isInitial, pending, path)

	// A value Set found zero is put back to zero if anything below it failed: a pointer or container
	// the tag allocated, a struct decoded partway, or what an unmarshaler wrote before it rejected the
	// tag. Left as it was, it would no longer be zero, and a second Set would skip the tag that failed.
	// This is not a defer in fillField: it has too many returns for the compiler to open-code one, and
	// the defer it builds instead slows every zero field Set fills.
	if err != nil && isInitial {
		field.Set(reflect.Zero(field.Type()))
	}
	return taken, err
}

// fillField is setField for a field it does not leave alone: it applies the tag if isInitial, which
// reports whether the field is zero, and descends into the field.
func fillField(field reflect.Value, tag fieldTag, isInitial bool, pending *pendingDefault, path *walking) (bool, error) {
	// Kept as a local because the parsing below reads better against a plain name.
	defaultVal := tag.value

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
					return false, fmt.Errorf("field %s: default %q recurses without end", tag.fieldName, defaultVal)
				}
			}
			pending = &pendingDefault{typ: field.Type(), value: defaultVal, outer: pending}
		}

		unmarshaled, unmarshalErr := unmarshalByInterface(field, defaultVal)
		if unmarshaled {
			return true, nil
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
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(val).Convert(field.Type()))
		case reflect.Int:
			val, err := strconv.ParseInt(defaultVal, 0, strconv.IntSize)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(int(val)).Convert(field.Type()))
		case reflect.Int8:
			val, err := strconv.ParseInt(defaultVal, 0, 8)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(int8(val)).Convert(field.Type()))
		case reflect.Int16:
			val, err := strconv.ParseInt(defaultVal, 0, 16)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(int16(val)).Convert(field.Type()))
		case reflect.Int32:
			val, err := strconv.ParseInt(defaultVal, 0, 32)
			if err != nil {
				return false, parseErr(err)
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
			// When both fail, both errors are reported, since the tag may have been meant for either,
			// unless the type's own unmarshaler rejected it first: parseErr reports that rejection.
			if val, err := time.ParseDuration(strings.TrimSpace(defaultVal)); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			} else if val, intErr := strconv.ParseInt(defaultVal, 0, 64); intErr == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			} else {
				return false, parseErr(fmt.Errorf("%w; %w", err, intErr))
			}
		case reflect.Uint:
			val, err := strconv.ParseUint(defaultVal, 0, strconv.IntSize)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(uint(val)).Convert(field.Type()))
		case reflect.Uint8:
			val, err := strconv.ParseUint(defaultVal, 0, 8)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(uint8(val)).Convert(field.Type()))
		case reflect.Uint16:
			val, err := strconv.ParseUint(defaultVal, 0, 16)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(uint16(val)).Convert(field.Type()))
		case reflect.Uint32:
			val, err := strconv.ParseUint(defaultVal, 0, 32)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(uint32(val)).Convert(field.Type()))
		case reflect.Uint64:
			val, err := strconv.ParseUint(defaultVal, 0, 64)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(val).Convert(field.Type()))
		case reflect.Uintptr:
			val, err := strconv.ParseUint(defaultVal, 0, strconv.IntSize)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(uintptr(val)).Convert(field.Type()))
		case reflect.Float32:
			val, err := strconv.ParseFloat(defaultVal, 32)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(float32(val)).Convert(field.Type()))
		case reflect.Float64:
			val, err := strconv.ParseFloat(defaultVal, 64)
			if err != nil {
				return false, parseErr(err)
			}
			field.Set(reflect.ValueOf(val).Convert(field.Type()))
		case reflect.String:
			field.Set(reflect.ValueOf(defaultVal).Convert(field.Type()))

		case reflect.Slice:
			ref := reflect.New(field.Type())
			ref.Elem().Set(reflect.MakeSlice(field.Type(), 0, 0))
			if defaultVal != "" && defaultVal != "[]" {
				if err := json.Unmarshal([]byte(defaultVal), ref.Interface()); err != nil {
					return false, parseErr(err)
				}
			}
			field.Set(ref.Elem().Convert(field.Type()))
		case reflect.Map:
			ref := reflect.New(field.Type())
			ref.Elem().Set(reflect.MakeMap(field.Type()))
			if defaultVal != "" && defaultVal != "{}" {
				if err := json.Unmarshal([]byte(defaultVal), ref.Interface()); err != nil {
					return false, parseErr(err)
				}
			}
			field.Set(ref.Elem().Convert(field.Type()))
		case reflect.Struct:
			if defaultVal != "" && defaultVal != "{}" {
				if err := json.Unmarshal([]byte(defaultVal), field.Addr().Interface()); err != nil {
					return false, parseErr(err)
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
				return false, parseErr(unmarshalErr)
			}
		}
	}

	switch field.Kind() {
	case reflect.Pointer:
		if isInitial || field.Elem().Kind() == reflect.Struct {
			taken, err := setField(field.Elem(), tag, pending, path)
			if err != nil {
				return false, err
			}

			// A struct pointee's setter is not called here: Set has called it as the recursion
			// finished, or skipped it because it is promoted, because an unmarshaler took the tag,
			// or because the struct is already being walked above. Anything else behind a pointer
			// the tag allocated gets no setter call but this one, and none when UnmarshalText or
			// UnmarshalJSON was handed the tag and took it, as a struct gets none then.
			if field.Elem().Kind() != reflect.Struct && !taken {
				callSetter(field.Interface())
			}
			return taken, nil
		}
	case reflect.Struct:
		if err := set(field.Addr().Interface(), pending, path); err != nil {
			return false, err
		}
	case reflect.Slice:
		if !canHoldDefaults(field.Type().Elem()) {
			break
		}
		here := walking{ptr: field.UnsafePointer(), typ: field.Type(), len: field.Len(), outer: path}
		if here.repeats() {
			break
		}
		for j := 0; j < field.Len(); j++ {
			if _, err := setField(field.Index(j), fieldTag{fieldName: tag.fieldName}, pending, &here); err != nil {
				return false, err
			}
		}
	case reflect.Map:
		if !canHoldDefaults(field.Type().Elem()) {
			break
		}
		here := walking{ptr: field.UnsafePointer(), typ: field.Type(), outer: path}
		if here.repeats() {
			break
		}
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

				if _, err := setField(v, fieldTag{fieldName: tag.fieldName}, pending, &here); err != nil {
					return false, err
				}

				if !originalIsPtr {
					field.SetMapIndex(e, v)
				}
			}
		}
	}

	return false, nil
}

// canHoldDefaults reports whether a slice element or map value of type t is of a kind the walk
// descends into. Such a value has no tag of its own, so descending into it is all the walk can do,
// and it descends only into a struct, a pointer, a slice or a map: setField stops a slice element
// of any other kind at shouldInitializeField, and the map loop's kind switch has no case for a map
// value of one. An array or an interface may hold something with defaults, but neither is descended
// into (TestSet_ArraysAreLeftAlone pins the array case), so a slice or map of any other kind would
// be walked entry by entry to change nothing. Keep this list in step with shouldInitializeField and
// the map loop's switch. The kind alone decides, so a pointer counts whatever it points to.
func canHoldDefaults(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct, reflect.Pointer, reflect.Slice, reflect.Map:
		return true
	}
	return false
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
// any tag: a struct is always descended into, as is a pointer the caller already allocated to a
// struct, and a container the caller already filled has elements to recurse into. Whether a tag is
// present is the caller's business.
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
