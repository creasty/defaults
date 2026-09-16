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
// A value no more than scanDepth entries down is checked by scanning the entries above it, which
// allocates nothing. Scanning further down would make a walk n values deep take on the order of n²/2
// comparisons, so a value below scanDepth is looked up in a pathIndex instead, which the first such
// check makes.
type walking struct {
	ptr   unsafe.Pointer
	typ   reflect.Type
	len   int
	outer *walking

	// depth is the number of entries above this one.
	depth int

	// index is the path's index, or nil while the walk has made none. The check that makes it stores it
	// into every entry then on the path; an entry declared after copies it, with depth, from the entry
	// above where set or fillField declares it. A method copying it would read it off one entry and
	// store it through a pointer into another, which the compiler takes for storing any pointer the first
	// entry holds, outer among them, and every entry on the path would move to the heap.
	index *pathIndex
}

// scanDepth is the deepest a value is checked by scanning the path: a scan that deep costs about as
// much as the index does for a value, and making the index, three allocations and about 9 KB, has
// paid for itself by about 100 values down. A walk no deeper than scanDepth makes no index.
const scanDepth = 64

// repeats reports whether the value w names is already being walked above it, on w.outer's path, if
// w is no more than scanDepth down. For a deeper w it reports false, and repeatsFar checks it: callers
// ask here.repeats() || here.repeatsFar(). The two are kept apart so that both inline. One method
// holding the scan and the call to repeatsInIndex would not, and the call to it for every value made
// Walk/slice/pointers/elements=1000 take 1.6 times as long in about half the runs.
func (w *walking) repeats() bool {
	if w.depth > scanDepth {
		return false
	}
	for p := w.outer; p != nil; p = p.outer {
		if p.ptr == w.ptr && p.typ == w.typ && p.len == w.len {
			return true
		}
	}
	return false
}

// repeatsFar is repeats for a w more than scanDepth down.
func (w *walking) repeatsFar() bool {
	return w.depth > scanDepth && w.repeatsInIndex()
}

// repeatsInIndex is repeatsFar's check. It makes the path's index if the walk has none yet, adds
// to it every entry above w that it does not hold, and then w, unless it already holds w's key.
func (w *walking) repeatsInIndex() bool {
	if w.index == nil {
		index := &pathIndex{path: make([]indexEntry, 256), heads: make([]int32, 256)}
		for p := w; p != nil; p = p.outer {
			p.index = index
		}
	}
	if w.index.held < w.depth {
		w.outer.addTo(w.index)
	}
	return !w.index.add(w.key())
}

// addTo adds w to x, after every entry above it that x does not hold yet. Each entry below scanDepth
// adds itself as it is checked, so the entries x lacks are within scanDepth of the top, and the calls
// to addTo stack no deeper.
func (w *walking) addTo(x *pathIndex) {
	if x.held < w.depth {
		w.outer.addTo(x)
	}
	x.add(w.key())
}

// leave takes w out of the path's index, if the index holds it, as w's walk returns. It is called on
// every return, a failed walk's too: a failure ends the whole Set, but an index still holding w would
// take a later value with w's key for a repeat.
func (w *walking) leave() {
	if w.index != nil && w.depth < w.index.held {
		w.index.pop()
	}
}

// key is w as the path's index holds it. A reflect.Type points to the one descriptor of its type, so
// the descriptor's address names the type, as == on Types does. UnsafePointer reads that address;
// Pointer would mark it as escaping.
func (w *walking) key() pathKey {
	return pathKey{ptr: uintptr(w.ptr), typ: uintptr(reflect.ValueOf(w.typ).UnsafePointer()), len: w.len}
}

// pathKey names an entry in the path's index by its address, its type and its length. They are held
// as uintptrs, not pointers: a pointer read off an entry and stored in the index would move every
// entry to the heap, as walking's index field explains. The entry keeps its value and its type alive
// while the index holds its key.
type pathKey struct {
	ptr, typ uintptr
	len      int
}

// hash mixes k into an int whose low bits pick a bucket.
func (k pathKey) hash() int {
	h := (uint64(k.ptr) ^ uint64(k.typ)*0x9e3779b97f4a7c15 ^ uint64(k.len)) * 0x9e3779b97f4a7c15
	return int(h ^ h>>32)
}

// pathIndex is the index of a path too deep to scan: the entries at the top of the path, down to some
// depth, and a hash table of their keys. Entries join and leave it as they do the path, at the bottom,
// so it holds them as a stack: path[d] is the entry d entries down, for each d below held.
//
// The table chains the entries whose keys share a bucket, deepest first. heads holds, for each
// bucket, one more than the position in path of its deepest entry, or 0 for none, and each entry's
// next does the same for the entry after it in the chain. The deepest entry of all heads its chain,
// so taking it out only moves its bucket's head on to its next.
//
// Neither slice is appended to or resliced; grow replaces both. The index is reached from an entry, so
// a slice read off it and stored back into it would be, to the compiler, a pointer read off that entry
// and stored on the heap, which would move every entry to the heap.
type pathIndex struct {
	path  []indexEntry
	heads []int32
	held  int
}

// indexEntry is an entry the path's index holds: its key, and the next entry in its bucket.
type indexEntry struct {
	key  pathKey
	next int32
}

// add adds an entry for k below the deepest x holds, and reports true, or reports false and adds
// nothing if x already holds k.
func (x *pathIndex) add(k pathKey) bool {
	if x.held == len(x.path) {
		x.grow()
	}
	b := k.hash() & (len(x.heads) - 1)
	for i := x.heads[b]; i != 0; i = x.path[i-1].next {
		if x.path[i-1].key == k {
			return false
		}
	}
	x.path[x.held] = indexEntry{key: k, next: x.heads[b]}
	x.held++
	x.heads[b] = int32(x.held)
	return true
}

// pop takes the deepest entry out of x.
func (x *pathIndex) pop() {
	x.held--
	e := x.path[x.held]
	x.heads[e.key.hash()&(len(x.heads)-1)] = e.next
}

// grow doubles x's room for entries and its buckets, as many of each and a power of two, and chains
// every entry again from the top down, so that each chain stays deepest first.
func (x *pathIndex) grow() {
	path := make([]indexEntry, 2*len(x.path))
	copy(path, x.path)
	x.path = path
	x.heads = make([]int32, len(path))
	for i := 0; i < x.held; i++ {
		b := path[i].key.hash() & (len(x.heads) - 1)
		path[i].next = x.heads[b]
		x.heads[b] = int32(i + 1)
	}
}

// Set fills the fields of the struct ptr points to from their `default` tags, and descends into
// structs, pointers, slices and maps to do the same below. A field is written only while it holds
// its zero value, and a nil map, slice or pointer is allocated only by a tag.
//
// A value reachable by more than one path is filled on each. Where data with a cycle leads back to a
// value still being walked, Set goes no further, so a cycle ends.
//
// A struct held as a map value is not addressable, so Set fills a copy and stores it back under its
// key whether or not anything changed: no other goroutine may read that map while Set runs. No
// other map value is stored back. A slice or map held as a map value is filled through, and so is
// a pointer to a struct, slice or map, but a pointer to anything else, such as a **T, is not.
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
	// below reaches the same struct as a plain *T. The struct at the top of the path has nothing above
	// it to repeat, so only one below it is checked.
	here := walking{ptr: p, typ: t, outer: path}
	if path != nil {
		here.depth, here.index = path.depth+1, path.index
		if here.repeats() || here.repeatsFar() {
			return nil
		}
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
			here.leave()
			return err
		}
	}
	here.leave()

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
			// An empty tag fails both as well, and parseErr would report nothing for it, so the
			// errors are joined only for a tag that is not empty: the join formats both messages,
			// which every empty tag would otherwise pay for.
			if val, err := time.ParseDuration(strings.TrimSpace(defaultVal)); err == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			} else if val, intErr := strconv.ParseInt(defaultVal, 0, 64); intErr == nil {
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			} else if defaultVal != "" {
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
		here := walking{
			ptr: field.UnsafePointer(), typ: field.Type(), len: field.Len(),
			outer: path, depth: path.depth + 1, index: path.index,
		}
		if here.repeats() || here.repeatsFar() {
			break
		}
		for j := 0; j < field.Len(); j++ {
			if _, err := setField(field.Index(j), fieldTag{fieldName: tag.fieldName}, pending, &here); err != nil {
				here.leave()
				return false, err
			}
		}
		here.leave()
	case reflect.Map:
		if !canHoldDefaults(field.Type().Elem()) {
			break
		}
		here := walking{
			ptr: field.UnsafePointer(), typ: field.Type(),
			outer: path, depth: path.depth + 1, index: path.index,
		}
		if here.repeats() || here.repeatsFar() {
			break
		}
		for _, e := range field.MapKeys() {
			v := field.MapIndex(e)

			// A pointer value is written through, so it needs no copy. Any other map value is not
			// addressable, so a struct, slice or map value is filled as a copy, and a struct copy is
			// stored back under its key, changed or not: a write to the map, as the Set doc warns.
			//
			// A slice or map copy is not stored back. It is a header over the same array or table as
			// the value in the map, which the walk fills in place without writing the header: with no
			// tag, setField leaves an empty header alone and walks a non-empty one, which is not zero,
			// without parsing it, and it resets only a value that was zero. So the copy still equals
			// what it was copied from, and storing it could only undo a store or delete that a
			// SetDefaults called in the walk made under this key.
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
					here.leave()
					return false, err
				}

				if !originalIsPtr && v.Kind() == reflect.Struct {
					field.SetMapIndex(e, v)
				}
			}
		}
		here.leave()
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
