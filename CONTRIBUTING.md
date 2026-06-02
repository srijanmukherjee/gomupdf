# Contributing to gomupdf

Thanks for your interest in improving gomupdf! Bug reports, fixes, docs, and
features are all welcome.

## Prerequisites

gomupdf is a cgo binding over the MuPDF C library, so you need a C toolchain and
MuPDF (library **and** headers) installed, with `CGO_ENABLED=1` (the default for
native builds). See the [Requirements](./README.md#requirements) section of the
README for per-OS install instructions.

## Building and testing

```sh
go build ./...
go test ./...          # runs against the committed fixtures in testdata/
go vet ./...
gofmt -l .             # should print nothing
```

A `Makefile` wraps these (`make`, `make test`, `make cover`). Test coverage is
gated in CI at **80%**; please keep new code covered.

## Project layout

- **`gomupdf`** (root package) — the core MuPDF binding: open/read/extract/
  render/write. All cgo lives here.
- **`geometry/`** — standalone 2D primitives (Point, Rect, IRect, Matrix, Quad).
- **`experimental/`** — a higher-level, ergonomic extraction layer. Its API is
  explicitly experimental and may change between releases.

## Submitting changes

1. For anything non-trivial, open an issue first so we can agree on the approach.
2. Keep pull requests focused on a single change.
3. Before opening a PR, make sure:
   - `gofmt` and `go vet ./...` are clean,
   - `go test ./...` passes,
   - coverage is maintained (≥ 80%),
   - every exported symbol has a doc comment (godoc style),
   - the README/docs are updated if behavior changed.
4. Fill in the pull request template.

## License

By contributing, you agree that your contributions are licensed under the
project's **AGPL-3.0** license (inherited from MuPDF).
