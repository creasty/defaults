# defaults

A reflection-based library that fills struct fields from `default:"..."` tags. Narrow scope by
design — read the design principles in
[#61](https://github.com/creasty/defaults/issues/61) before widening it.

## Commands

- `make test` — `go test -race -shuffle=on ./...`
- `make cover` — the same, plus a coverage profile and the per-function table
- `make lint` / `make fmt` — golangci-lint v2, configured in `.golangci.yml`

`go.mod` declares the oldest supported Go release and the CI matrix tests that version plus every
currently supported one. Keep the two in step.

## Tests

`package defaults_test` — black box, public API only, one file per behavior under test.

- Declare each test's struct inside the test function. Package-level types only where a method is
  required (`SetDefaults`, `UnmarshalText`, `UnmarshalJSON`), named with a file-unique prefix.
- No package-level `var`s: the suite stays order-independent, which `-shuffle=on` enforces.
- Keep test structs under ~6 fields, and don't share one across test functions unless it carries a
  method. The suite this replaced hung every assertion off a single ~120-field struct.
- Statement coverage of the package is 100%. Keep it there.

## Behavior changes

Today's behavior is pinned by tests, including the parts that look wrong — those carry a `// QUIRK`
or `// BUG` comment with a link. Changing behavior means flipping the pinned test in the same
commit, so the diff records the decision instead of burying it.
