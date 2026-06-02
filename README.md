# gomupdf

[![CI](https://github.com/srijanmukherjee/gomupdf/actions/workflows/ci.yml/badge.svg)](https://github.com/srijanmukherjee/gomupdf/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/srijanmukherjee/gomupdf/graph/badge.svg)](https://codecov.io/gh/srijanmukherjee/gomupdf)
[![Go Reference](https://pkg.go.dev/badge/github.com/srijanmukherjee/gomupdf.svg)](https://pkg.go.dev/github.com/srijanmukherjee/gomupdf)
[![Go Report Card](https://goreportcard.com/badge/github.com/srijanmukherjee/gomupdf)](https://goreportcard.com/report/github.com/srijanmukherjee/gomupdf)

A focused [cgo](https://pkg.go.dev/cmd/cgo) binding over the [MuPDF](https://mupdf.com/core)
C core for **reading, extracting, rendering, and writing PDF documents** from Go.

The API is compact and idiomatic: open a (possibly encrypted) PDF from a file or
from memory, then pull reading-order text, positioned word/span geometry, tables,
images, vector drawings, links, and the outline from each page. Pages render to
rasters, and new documents can be assembled, modified, and saved (optionally
encrypted).

## Requirements

This package links against the MuPDF C library via cgo, so a C toolchain and
MuPDF (shared library **and** development headers) must be present at build
time. `CGO_ENABLED=1` is required (it is on by default for native builds).

### Installing MuPDF

**macOS (Homebrew)**

```sh
brew install mupdf
```

Homebrew installs into `/opt/homebrew` (Apple Silicon) or `/usr/local` (Intel);
the default cgo flags target `/opt/homebrew`. On Intel Macs, point cgo at the
right prefix (see [Custom install locations](#custom-install-locations)).

**Debian / Ubuntu**

```sh
sudo apt-get install libmupdf-dev mupdf-tools
```

**Fedora / RHEL**

```sh
sudo dnf install mupdf-devel
```

**Arch Linux**

```sh
sudo pacman -S libmupdf
```

**From source** (any platform, when no package is available)

```sh
git clone --recursive https://github.com/ArtifexSoftware/mupdf.git
cd mupdf
make HAVE_X11=no HAVE_GLUT=no prefix=/usr/local install
```

> Windows is not covered by the default cgo flags; build with a MinGW/MSYS2
> toolchain and set `CGO_CFLAGS` / `CGO_LDFLAGS` to your MuPDF install (see
> below).

### Custom install locations

The default cgo flags (declared in `mupdf.go`) target the standard prefixes:
`-I/opt/homebrew/include -L/opt/homebrew/lib` on darwin and the system include
paths on linux, linking `-lmupdf` (and `-lmupdf-third` on linux). If your MuPDF
lives elsewhere, supplement the flags via environment variables instead of
editing the source:

```sh
export CGO_CFLAGS="-I/your/prefix/include"
export CGO_LDFLAGS="-L/your/prefix/lib -lmupdf -lmupdf-third"
go build ./...
```

### Verify

```sh
go install github.com/srijanmukherjee/gomupdf/...@latest  # should compile cleanly
```

A linker error such as `ld: library 'mupdf' not found` means MuPDF is not
installed or not on the library path — install it or set `CGO_LDFLAGS` as above.

## Install

```sh
go get github.com/srijanmukherjee/gomupdf
```

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/srijanmukherjee/gomupdf"
)

func main() {
	doc, err := gomupdf.Open("document.pdf")
	if err != nil {
		log.Fatal(err)
	}
	defer doc.Close()

	if doc.NeedsPass() && !doc.Authenticate("secret") {
		log.Fatal("wrong password")
	}

	for i, page := range doc.Pages() {
		text, err := page.GetText()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("page %d:\n%s\n", i, text)
	}
}
```

Convenience openers handle authentication in one step and return an error on a
wrong password:

```go
doc, err := gomupdf.OpenWithPassword("document.pdf", password)
doc, err := gomupdf.OpenStreamWithPassword(pdfBytes, password)
```

## Text & geometry

All geometry is in PDF points, origin top-left, y growing downward.

```go
text, _   := doc.Text()          // whole document, pages joined by form feed
pages, _  := doc.TextByPage()    // per-page text
lines, _  := doc.AllLines()      // flat line stream, soft hyphens stripped, blanks dropped

blocks, _ := page.StructuredText() // []Block → []Span with Rect, font, baseline
spans, _  := page.Spans()          // flattened text fragments in document order
words, _  := page.Words()          // word boxes with character-level precision
```

`page.StructuredJSON()` returns the raw structured-text JSON for callers that
want to traverse the block/line/char tree themselves.

## Search, outline & links

```go
quads, _ := page.Search("invoice")     // oriented quads for each hit
rects, _ := page.SearchRects("invoice") // axis-aligned rects
toc, _   := doc.TOC()                   // flat, depth-first outline
links, _ := page.Links()                // clickable rects + target URIs
```

## Tables

`page.FindTables` detects tables with either a word-alignment strategy or a
vector-ruling strategy:

```go
tables, _ := page.FindTables() // default: word-alignment ("text") strategy
tables, _ := page.FindTables(gomupdf.TableSettings{Strategy: gomupdf.StrategyLines})

for _, t := range tables {
	fmt.Println(t.NumRows(), t.NumCols(), t.Rows)
}
```

## Rendering & images

```go
pm, _ := page.Pixmap(gomupdf.PixmapOptions{Zoom: 2.0}) // 144 DPI RGB raster
png, _ := pm.PNG()
_ = pm.SavePNG("page.png")

imgs, _ := page.GetImages()        // placement + source metadata
img, _  := page.ExtractImage(0)    // original encoded bytes (JPEG/PNG/...)
paths, _ := page.GetDrawings()     // vector fill/stroke paths
```

## Writing & assembly

```go
doc, _ := gomupdf.NewPDF()
defer doc.Close()
doc.NewPage(595, 842)              // A4 in points
p, _ := doc.LoadPage(0)
p.InsertText(72, 800, 12, "Hello")
p.InsertImage(geometry.NewRect(0, 0, 200, 200), imageBytes)
doc.InsertPDF(otherPDFBytes, "")   // append all pages of another document

data, _ := doc.SaveBytes(true)                       // serialize (with garbage collection)
data, _ := doc.SaveEncryptedBytes("user", "owner")   // AES-256 encrypted
```

## Subpackages

- [`geometry`](./geometry) — the 2D primitives (`Point`, `Rect`, `IRect`,
  `Matrix`, `Quad`) used throughout the API.
- [`experimental`](./experimental) — a higher-level, ergonomic layer:
  lifecycle-managed documents, a row/word model for layout-aware extraction,
  region and key/value queries, located regex search, and helpers for rendering,
  image extraction, and merging PDFs and images.

## Feature support

Legend: ✅ supported · 🟡 partial · ⬜ not yet

| Area | Capability | Status |
|---|---|---|
| Read | Open / stream / password / metadata | ✅ |
| Read | Page count, load, iterate | ✅ |
| Read | Plain & structured text, char-level words | ✅ |
| Read | Geometry primitives (`geometry` pkg) | ✅ |
| Read | Text search (quads & rects) | ✅ |
| Read | Tables — word-alignment & vector-ruling strategies | ✅ |
| Read | Textbox / line-break extraction | ⬜ |
| Structure | Outline (TOC) | ✅ |
| Structure | Page links | ✅ |
| Structure | Page labels, page boxes, embedded files | ⬜ |
| Render | Pixmap → raster (PNG, image.Image, pixel access) | ✅ |
| Render | Extract images + placement bbox | ✅ |
| Render | Vector drawings / paths | ✅ |
| Render | OCR, image masks | ⬜ |
| Write | New PDF / page / delete / merge | ✅ |
| Write | Insert text & images | ✅ |
| Write | Save (full rewrite) & AES-256 encryption | ✅ |
| Write | Annotations | 🟡 rectangle only |
| Write | Widgets/forms, redaction, overlay, fonts | ⬜ |

## Concurrency & memory

A `Document` owns a single native context and is **not** safe for concurrent
use; all of its methods are serialized internally by a mutex. To process
documents in parallel, open one `Document` per goroutine. Always `Close()` a
document when done (a finalizer is only a backstop). The in-memory bytes backing
a streamed PDF are held for the document's lifetime.

## Acknowledgements

- **[MuPDF](https://mupdf.com)** by [Artifex Software](https://artifex.com) is
  the C library that does the heavy lifting — parsing, rendering, and text
  extraction. This package is a thin Go layer over it; all credit for the
  underlying PDF engine belongs to the MuPDF authors.
- **[PyMuPDF](https://github.com/pymupdf/PyMuPDF)** (the Python MuPDF binding)
  and **[pdfplumber](https://github.com/jsvine/pdfplumber)** are the prior art
  that shaped this library's extraction and table-detection design. The
  word-alignment table strategy follows the well-trodden approach those projects
  established. Thank you to their authors and maintainers.

## License

MuPDF is distributed under the **GNU AGPL-3.0**, and this package links against
it, so the same license applies (see [LICENSE](./LICENSE)). Review its
obligations before distributing binaries built with this package.
