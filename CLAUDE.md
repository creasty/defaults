# defaults

A reflection-based library that fills struct fields from `default:"..."` tags. Narrow scope by
design — read the design principles in
[#61](https://github.com/creasty/defaults/issues/61) before widening it.

## Commands

- `make test` — `go test -race -shuffle=on ./...`
- `make cover` — the same, plus a coverage profile and the per-function table
- `make bench` — the benchmarks, without `-race` or `-shuffle=on`: both distort the numbers
- `make bench-smoke` — every benchmark once (`-benchtime 1x`), as CI runs it on each Go version: it
  fails when a benchmark stops compiling or fails its check, and records no timings
- `make bench-compare BASE=<ref>` — `BASE` (default `origin/master`) against the working tree,
  uncommitted changes included: `ROUNDS` (6) interleaved rounds of `BENCH` (`.`) at `BENCHTIME`
  (400ms), both sides built with the working tree's `benchmark_test.go`, then benchstat; raw results
  stay in `.bench-compare/`
- `make lint` / `make fmt` — golangci-lint v2, configured in `.golangci.yml`

`go.mod` declares the oldest supported Go release and the CI matrix tests that version plus every
currently supported one. Keep the two in step.

## Tests

`package defaults_test` — black box, public API only, one file per behavior under test. The files
stay beside the source, not in a subdirectory: coverage of `.` and the `Example*` functions
pkg.go.dev renders both depend on the tests living in the package's own directory.

Machinery the public API reaches only through `Set` — the promoted-method check, the unmarshaler
handoff, the path check — lives in a file of its own, with its tests beside it in `package defaults`
rather than `package defaults_test`, since nothing else can reach it. Those files are the only
exception to the black-box rule above. Keep them in the root package: behind an `internal/` package
the walk's hot path crosses a package boundary, which measured 1.7% to 4.1% slower across seven of
the eleven `Walk/slice` and `Walk/map` rows, though the inlining and escape decisions are identical.

- Declare each test's struct inside the test function. Package-level types only where a method is
  required (`SetDefaults`, `UnmarshalText`, `UnmarshalJSON`), named with a file-unique prefix —
  except in `example_test.go`, whose types keep the plain names readers see on pkg.go.dev.
- No package-level `var`s: the suite stays order-independent, which `-shuffle=on` enforces.
- Keep test structs under ~6 fields, and don't share one across test functions unless it carries a
  method. The suite this replaced hung every assertion off a single ~120-field struct. A struct
  that lists one field per kind, because each kind has its own parse branch, is the exception.
- Statement coverage of the package is 100%. Keep it there.

## Benchmarks

`benchmark_test.go` follows the test rules above, and these as well:

- Every case calls `b.ReportAllocs`. A case that calls `Set` does its setup and one check of the
  value it measures before `b.ResetTimer`, and does the same work each iteration: a zero value reset
  in place and filled again, or `Set` on a value `Set` leaves as it is — never a value that changes
  across iterations. A `CanUpdate` case has no setup, so it checks inside its loop.
- The timed loop does nothing but that reset and the measured call: no `b.Helper`, no `b.StopTimer`.
- The file stands alone: `make bench-compare` builds it with no other test file, so it uses only the
  public API. Every case builds and passes its checks on every commit from v1.10.0 on, so no input
  may be one those commits cannot handle, such as a cycle. Move that floor only on purpose, here.
  Older commits need a narrower `BENCH` (v1.9.0 fails `BenchmarkFail/unmarshaler`'s check), and
  those whose `go.mod` is below Go 1.18 cannot build the file.
- Names have no spaces, `_` inside a segment, and a scaled family's `key=value` last.
  `BenchmarkSet/scalars`, `BenchmarkSet/composites`, `BenchmarkCanUpdate/scalar` and
  `BenchmarkCanUpdate/struct` keep their names and loops, for comparison with numbers already
  published.

## Behavior changes

Today's behavior is pinned by tests, including the parts that look wrong — those carry a `// QUIRK`
or `// BUG` comment, with a link to the issue or PR behind it when one exists. Changing behavior
means flipping the pinned test in the same commit, so the diff records the decision instead of
burying it.
