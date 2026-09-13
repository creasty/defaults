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
  - Implementing [`encoding.TextUnmarshaler`](https://pkg.go.dev/encoding#TextUnmarshaler), which
    takes precedence over `defaults.Setter` -- the tag is handed to `UnmarshalText` and
    `SetDefaults` is not called
- Preserves non-initial values from being reset with a default value

The API is three functions: `Set`, `MustSet` (the same, but panicking), and `CanUpdate`. Runnable
examples for each are on [pkg.go.dev](https://pkg.go.dev/github.com/creasty/defaults#pkg-examples),
and [the package example](https://pkg.go.dev/github.com/creasty/defaults#example-package) walks the
composite cases in one program.


## Behavior

### Zero values

A field is written only while it still holds its type's zero value. No scalar type can tell an
unspecified value from its zero value — an `int` left alone is `0`, a `string` is `""`, a `bool` is
`false` — so a zero the caller set on purpose is indistinguishable from one never set, and the
default replaces it.

Use a pointer where that distinction matters. `nil` means unspecified, and a pointer to the zero
value is preserved: `*bool` is the way to let `false` survive a `default:"true"`.

```go
type Feature struct {
	Enabled bool  `default:"true"` // a caller's false is replaced
	Verbose *bool `default:"true"` // a caller's &false survives
}
```

`default:"-"` opts a field out entirely: it is neither parsed nor recursed into. That is how a field
gets its value from a `SetDefaults` method instead of a tag.

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


## Design principles

**Keep it simple.** The scope is narrow by design — read a tag, fill a field — and is meant to stay
that way. Adjacent concerns belong in adjacent libraries.

**Stay comprehensive.** Broad type support and high test coverage are what make the library
trustworthy. Statement coverage is 100% and CI enforces it, and every supported type has a test
pinning its behavior — including the behavior that looks wrong, which is pinned with a comment
saying so rather than left to be rediscovered.


## License

[MIT](./LICENSE)
