# defaults

[![CI](https://img.shields.io/github/actions/workflow/status/creasty/defaults/ci.yml?branch=master&label=CI)](https://github.com/creasty/defaults/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/creasty/defaults/branch/master/graph/badge.svg)](https://codecov.io/gh/creasty/defaults)
[![Go Reference](https://pkg.go.dev/badge/github.com/creasty/defaults.svg)](https://pkg.go.dev/github.com/creasty/defaults)
[![GitHub release](https://img.shields.io/github/release/creasty/defaults.svg)](https://github.com/creasty/defaults/releases)
[![License](https://img.shields.io/github/license/creasty/defaults.svg)](./LICENSE)

Initialize structs with default values

```go
type Server struct {
	Host    string            `default:"localhost"`
	Port    int               `default:"8080"`
	Timeout time.Duration     `default:"30s"`
	Tags    []string          `default:"[\"web\"]"`
	Labels  map[string]string `default:"{\"env\": \"dev\"}"`
}

var server Server
if err := defaults.Set(&server); err != nil {
	panic(err)
}
// localhost:8080 30s [web] map[env:dev]
```


## Install

```console
$ go get github.com/creasty/defaults
```


## Features

- Supports almost all kind of types
  - Scalar types
    - `int/8/16/32/64`, `uint/8/16/32/64`, `float32/64`
    - `uintptr`, `bool`, `string`
  - Complex types
    - `map`, `slice`, `struct`
  - Nested types
    - `map[K1]map[K2]Struct`, `[]map[K1][]Struct`
  - Aliased types
    - `time.Duration`
    - e.g., `type Enum string`
  - Pointer types
    - e.g., `*SampleStruct`, `*int`
- Recursively initializes fields in a struct
- Dynamically sets default values by:
  - Implementing the [`defaults.Setter`](./setter.go) interface, or
  - Implementing [`encoding.TextUnmarshaler`](https://pkg.go.dev/encoding#TextUnmarshaler) or
    [`json.Unmarshaler`](https://pkg.go.dev/encoding/json#Unmarshaler), which takes precedence
    over `defaults.Setter` -- a field's own tag is handed to `UnmarshalText` (or `UnmarshalJSON`),
    and when that takes it, `SetDefaults` is not called for the field (see
    [Unmarshalers](#unmarshalers))
- Preserves non-initial values from being reset with a default value

The API is three functions: `Set`, `MustSet` (the same, but panicking), and `CanUpdate`. Runnable
examples for `Set`, `CanUpdate` and `Setter` are on
[pkg.go.dev](https://pkg.go.dev/github.com/creasty/defaults#pkg-examples), and
[the package example](https://pkg.go.dev/github.com/creasty/defaults#example-package), which calls
`MustSet`, walks the composite cases in one program.


## Behavior

### Zero values

A field is written only while it still holds its type's zero value. No scalar type can tell an
unspecified value from its zero value — an `int` left alone is `0`, a `string` is `""`, a `bool` is
`false` — so a zero the caller set on purpose is indistinguishable from one never set, and the
default replaces it.

Use a pointer where that distinction matters. `nil` means unspecified, and a pointer to a zero
scalar is preserved: `*bool` is the way to let `false` survive a `default:"true"`. A pointer to a
struct is descended into like the struct itself, though, so the struct's zero fields still get
their defaults. A pointer the caller allocated to anything else is not: neither its tag nor the walk
goes past it, so the elements behind a caller's `*[]T` or `*map[K]T`, and the struct behind a `**T`,
get no defaults, and a default that would fail there is not reported. The same `*[]T` or `*map[K]T`
held as a map value is descended into, though a `**T` held as one is not.

```go
type Feature struct {
	Enabled bool  `default:"true"` // a caller's false is replaced
	Verbose *bool `default:"true"` // a caller's &false survives
}
```

`default:"-"` opts a field out entirely: it is neither parsed nor recursed into. That is how a field
gets its value from a `SetDefaults` method instead of a tag.

### Integers

An integer tag is a Go integer literal: `0x`, `0o` and `0b` prefixes and `_` separators all work,
and so does the legacy octal of a leading `0`, so `default:"0644"` is 420 and `default:"08080"` is
an error. Slice and map tags are JSON instead, where a number rejects a leading zero, and an integer
map key, a string in JSON, is read as decimal: `{"010": ...}` is key 10.

### Durations

A `time.Duration` takes anything [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration)
accepts. Every field that is an `int64` underneath is parsed the same way, though, so a
`time.Duration` takes a bare integer and a plain `int64` takes a duration string:

```go
type Job struct {
	Timeout time.Duration `default:"1"`  // 1ns, not 1s
	Retries int64         `default:"5m"` // 300000000000, not an error
}
```

This is deliberate: a type defined from `time.Duration` keeps no trace of it at runtime, so parsing
durations only for `time.Duration` itself would stop such types from taking `"10s"`.

### Unmarshalers

The precedence of `UnmarshalText` and `UnmarshalJSON` over `SetDefaults` covers a tag handed to
them and taken. A tag they reject is parsed by the field's kind instead, so a struct parsed that way
gets its own field tags and `SetDefaults` as usual. `{}` and `[]` are never handed to `UnmarshalJSON`
directly: `{}` allocates an empty struct or map and `[]` an empty slice, while `[]` on a struct or map
and `{}` on a slice are parsed through `encoding/json`, which may call `UnmarshalJSON` on the way
without that counting as taking the tag. A value built from a parent's JSON tag, such as an element
of a slice, is filled like any other too, whatever unmarshaler `encoding/json` ran while decoding
it.

### Shared and cyclic values

`Set` keeps track of the values on the path it is walking, not of every value it has walked. A value
reachable by more than one path -- two pointers to one struct, a map two fields hold -- is filled on
each path, and its `SetDefaults` is called on each, so a setter that is not idempotent applies itself
more than once, and a chain of values each shared by two pointers takes time that doubles with every
link. Data with a cycle, such as a pointer back up to a parent, ends: where it leads back to a value
still being walked, `Set` goes no further, and the walk already under way finishes that value.

### Maps

A struct held as a map value cannot be filled in place, so `Set` fills a copy and stores it back
under its key: every such struct it walks, whether or not anything changed. That store is a write to
the map, so do not call `Set` while another goroutine reads a map of structs it walks, even one with
nothing left to fill. No other map value is stored back, so `Set` only reads a map of anything else.
A slice or map held as a map value is filled through, as is a pointer to a struct, slice or map; a
pointer to anything else, such as a `**T`, is not descended into.

### Errors

`Set` stops at the first field whose default fails and returns an error naming it. A value `Set`
found zero on the way to that default is put back to zero, whatever part of the default it had
taken, so calling `Set` again fails again. Fields filled elsewhere before the failure keep their
defaults.


## Design principles

**Keep it simple.** The scope is narrow by design — read a tag, fill a field — and is meant to stay
that way. Adjacent concerns belong in adjacent libraries.

**Stay comprehensive.** Broad type support and high test coverage are what make the library
trustworthy. Statement coverage is 100% and CI enforces it, and every supported type has a test
pinning its behavior — including the behavior that looks wrong, which is pinned with a comment
saying so rather than left to be rediscovered.


## License

[MIT](./LICENSE)
