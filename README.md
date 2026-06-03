<div align="center">

# gomupdf

**The fast, idiomatic way to read, render, and write PDFs in Go.**

A focused [cgo](https://pkg.go.dev/cmd/cgo) binding over the battle-tested
[MuPDF](https://mupdf.com/core) C core — text extraction, table detection,
rendering, and full document assembly, behind a small, Go-shaped API.

[![CI](https://github.com/srijanmukherjee/gomupdf/actions/workflows/ci.yml/badge.svg)](https://github.com/srijanmukherjee/gomupdf/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/srijanmukherjee/gomupdf/graph/badge.svg)](https://codecov.io/gh/srijanmukherjee/gomupdf)
[![Go Reference](https://pkg.go.dev/badge/github.com/srijanmukherjee/gomupdf.svg)](https://pkg.go.dev/github.com/srijanmukherjee/gomupdf)
[![Go Report Card](https://goreportcard.com/badge/github.com/srijanmukherjee/gomupdf)](https://goreportcard.com/report/github.com/srijanmukherjee/gomupdf)
[![Built with Claude Code](https://img.shields.io/badge/Built%20with-Claude%20Code-D97757?logo=anthropic&logoColor=white)](https://claude.com/claude-code)

</div>

```go
doc, _ := gomupdf.Open("invoice.pdf")
defer doc.Close()

text, _   := doc.Text()         // every page, in reading order
page, _   := doc.LoadPage(0)
tables, _ := page.FindTables()  // structured rows & columns

pm, _ := page.Pixmap()          // render the page…
_ = pm.SavePNG("page.png")      // …and save it as an image
```

---

## Why gomupdf?

- **One dependency, many formats.** PDF, XPS, EPUB, MOBI, FB2, CBZ, SVG, plain
  text, and raster images — all open through the same `Document` API.
- **Extraction that keeps geometry.** Plain text *and* word/span boxes,
  reading-order reconstruction, and table detection with two strategies
  (word-alignment and vector-ruling).
- **Real writing, not just reading.** Create documents, insert text/images,
  merge, set metadata/XMP, page labels, page boxes, and save with fine-grained
  options including AES-256 encryption and incremental updates.
- **Idiomatic Go.** `iter.Seq2` page ranges, typed geometry, `error` returns,
  deterministic `Close()`, no hidden globals.
- **Thin and honest.** A small layer over MuPDF — the credit for the engine
  belongs upstream; gomupdf gives it a clean Go surface.

## Table of contents

- [Install](#install) · [Requirements](#requirements)
- [60-second tour](#60-second-tour)
- **Cookbook** — [Open](#open-anything) · [Text & geometry](#text--geometry) ·
  [Tables](#tables) · [Search, outline & links](#search-outline--links) ·
  [Render & images](#render--images) · [Pages & boxes](#pages-rotation--boxes) ·
  [Metadata & labels](#metadata--page-labels) · [Write & assemble](#write--assemble) ·
  [Save options](#save-options)
- [API cheat-sheet](#api-cheat-sheet) · [Feature support](#feature-support)
- [Concurrency & memory](#concurrency--memory) · [Subpackages](#subpackages)
- [Acknowledgements](#acknowledgements) · [License](#license)

---

## Install

```sh
go get github.com/srijanmukherjee/gomupdf
```

### Requirements

gomupdf links against the MuPDF C library via cgo, so a **C toolchain** and
**MuPDF (shared library + development headers)** must be present at build time.
`CGO_ENABLED=1` is required (on by default for native builds).

| Platform | Install MuPDF |
|---|---|
| macOS (Homebrew) | `brew install mupdf` |
| Debian / Ubuntu | `sudo apt-get install libmupdf-dev mupdf-tools` |
| Fedora / RHEL | `sudo dnf install mupdf-devel` |
| Arch | `sudo pacman -S libmupdf` |

<details>
<summary><b>From source / custom prefix / Windows / troubleshooting</b></summary>

**From source** (any platform):

```sh
git clone --recursive https://github.com/ArtifexSoftware/mupdf.git
cd mupdf && make HAVE_X11=no HAVE_GLUT=no prefix=/usr/local install
```

**Custom install location.** The default cgo flags (in `mupdf.go`) target
`/opt/homebrew` on darwin and the system paths on linux, linking `-lmupdf`
(plus `-lmupdf-third` on linux). If MuPDF lives elsewhere, supplement via
environment variables instead of editing source:

```sh
export CGO_CFLAGS="-I/your/prefix/include"
export CGO_LDFLAGS="-L/your/prefix/lib -lmupdf -lmupdf-third"
go build ./...
```

**Windows** is not covered by the default flags; build with a MinGW/MSYS2
toolchain and set `CGO_CFLAGS` / `CGO_LDFLAGS` to your MuPDF install.

**`ld: library 'mupdf' not found`** means MuPDF is not installed or not on the
library path — install it or set `CGO_LDFLAGS` as above.

</details>

---

## 60-second tour

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
	defer doc.Close() // always Close; the finalizer is only a backstop

	// Encrypted? Unlock it (or use OpenWithPassword to do both in one call).
	if doc.NeedsPass() && !doc.Authenticate("secret") {
		log.Fatal("wrong password")
	}

	fmt.Println("pages:", doc.PageCount())

	for i, page := range doc.Pages() { // ranges via iter.Seq2
		text, err := page.GetText()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("=== page %d ===\n%s\n", i, text)
	}
}
```

> **Coordinates.** All geometry is in PDF points. The `geometry` package uses a
> top-left origin with y growing downward (`Rect{X0,Y0,X1,Y1}`).

---

## Cookbook

### Open anything

```go
// PDF fast paths
doc, _ := gomupdf.Open("file.pdf")                      // from a file
doc, _ := gomupdf.OpenStream(pdfBytes)                  // from memory
doc, _ := gomupdf.OpenWithPassword("file.pdf", "pw")    // open + authenticate
doc, _ := gomupdf.OpenStreamWithPassword(b, "pw")

// Any MuPDF-supported format (XPS, EPUB, MOBI, FB2, CBZ, SVG, TXT, images)
doc, _ := gomupdf.OpenAny("book.epub")                  // format from extension
doc, _ := gomupdf.OpenAnyStream(pngBytes, "png")        // explicit format hint

// Turn a non-PDF (or any) document into a writable PDF
pdf, _ := doc.ConvertToPDF()
defer pdf.Close()
```

Non-PDF documents open **read-only** (text, geometry, rendering all work); use
`ConvertToPDF` to get a writable PDF `Document`.

### Text & geometry

```go
text, _   := doc.Text()            // whole document, pages joined by '\f'
pages, _  := doc.TextByPage()      // []string, one per page
lines, _  := doc.AllLines()        // flat lines: soft-hyphens stripped, blanks dropped

words, _  := page.Words()          // word boxes with character-level precision
spans, _  := page.Spans()          // positioned fragments + font name/size
blocks, _ := page.StructuredText() // []Block → []Span tree with bbox & baseline
raw, _    := page.StructuredJSON() // the raw block/line/char JSON, your way
```

`Words` is the recommended input for layout work and table reconstruction — the
boxes are unions of per-character quads, not span approximations.

### Tables

Two strategies: **word-alignment** (`StrategyText`, the default) and
**vector-ruling** (`StrategyLines`, follows drawn gridlines).

```go
tables, _ := page.FindTables() // default word-alignment strategy
tables, _ := page.FindTables(gomupdf.TableSettings{
	Strategy:      gomupdf.StrategyLines,
	SnapTolerance: 3,
})

for _, t := range tables {
	fmt.Println(t.NumRows(), "x", t.NumCols())
	for _, row := range t.Rows {
		fmt.Println(row) // []string cells
	}
}
```

### Search, outline & links

```go
quads, _ := page.Search("invoice")       // oriented quads (great for highlights)
rects, _ := page.SearchRects("invoice")  // axis-aligned rects
toc, _   := doc.TOC()                     // flat, depth-first outline entries
links, _ := page.Links()                  // clickable rects + target URIs
```

### Render & images

```go
pm, _  := page.Pixmap(gomupdf.PixmapOptions{Zoom: 2.0}) // 144 DPI RGB raster
png, _ := pm.PNG()                                       // encode to PNG bytes
_ = pm.SavePNG("page.png")
img, _ := pm.Image()                                     // a stdlib image.Image
px := pm.Pixel(10, 10)                                   // raw component bytes

infos, _ := page.GetImages()       // each image's placement bbox + source dims
raw, _   := page.ExtractImage(0)   // original encoded bytes (JPEG/PNG/…)
paths, _ := page.GetDrawings()     // vector fill/stroke paths with color & width
```

### Pages, rotation & boxes

```go
n      := doc.PageCount()
page, _ := doc.LoadPage(0)

rot, _  := page.Rotation()           // 0, 90, 180, or 270
_ = page.SetRotation(90)             // multiples of 90, normalized

mb, _ := page.MediaBox()             // full physical page, unrotated points
cb, _ := page.CropBox()              // visible region (defaults to MediaBox)
_ = page.SetCropBox(geometry.Rect{X0: 10, Y0: 10, X1: 400, Y1: 600})
```

### Metadata & page labels

```go
// Read / write the standard info dictionary
meta, _ := doc.Metadata()            // map: title, author, producer, dates, …
_ = doc.SetMetadata(map[string]string{
	"title":  "Quarterly Report",
	"author": "Finance",
	// an empty value clears a field; unknown keys are ignored
})

// XMP packet (Catalog /Metadata stream)
xmp, _ := doc.XMP()
_ = doc.SetXMP(xmlPacket)
_ = doc.DeleteXMP()

// Logical page numbering: front-matter as i, ii, iii then body 1, 2, 3 …
_ = doc.SetPageLabels([]gomupdf.PageLabel{
	{StartPage: 0, Style: gomupdf.LabelRomanLower, Start: 1},
	{StartPage: 2, Style: gomupdf.LabelDecimal, Prefix: "", Start: 1},
})
label, _ := page.Label()             // resolved label, e.g. "ii" or "A-3"
rules, _ := doc.PageLabels()          // read the rules back
```

### Write & assemble

```go
doc, _ := gomupdf.NewPDF()
defer doc.Close()

doc.NewPage(595, 842)                 // A4 in points
p, _ := doc.LoadPage(0)
p.InsertText(72, 800, 12, "Hello, gomupdf")
p.InsertImage(geometry.NewRect(0, 0, 200, 200), imageBytes)

doc.InsertPDF(otherPDFBytes, "")      // append every page of another PDF
doc.DeletePage(3)
```

### Save options

```go
data, _ := doc.SaveBytes(true)                     // quick: serialize (+garbage)
_ = doc.EzSave("out.pdf")                          // sensible defaults to a file

// Fine-grained control
data, _ = doc.SaveBytesWithOptions(gomupdf.SaveOptions{
	Garbage:       3,     // 0–4 dead-object collection
	Deflate:       true,  // compress streams
	Clean:         true,  // sanitize content streams
	Linear:        true,  // "fast web view"
	ASCII:         true,  // escape binary to 7-bit
})

// AES-256 encryption
data, _ = doc.SaveBytesWithOptions(gomupdf.SaveOptions{
	Encrypt: true, UserPassword: "open-me", OwnerPassword: "full", Permissions: -1,
})
data, _ = doc.SaveEncryptedBytes("user", "owner")  // shorthand

// Incremental update (file/stream-opened docs): appends changes after the
// original bytes instead of rewriting the whole file.
_ = doc.SaveIncremental("out.pdf")
```

---

## API cheat-sheet

| Want to… | Call |
|---|---|
| Open a PDF | `Open` · `OpenStream` · `OpenWithPassword` |
| Open EPUB/XPS/image/… | `OpenAny` · `OpenAnyStream` |
| Convert to writable PDF | `doc.ConvertToPDF()` |
| Count / load / iterate pages | `doc.PageCount` · `doc.LoadPage` · `doc.Pages` |
| Extract text | `doc.Text` · `page.GetText` · `page.Lines` · `page.ExtractText` (clip+flags) |
| Get positioned text | `page.Words` · `page.Spans` · `page.StructuredText` · `page.RawDict`/`Dict` |
| Export markup | `page.HTML` · `page.XHTML` · `page.XML` |
| Find tables | `page.FindTables` → `table.ToMarkdown` / `table.Header` |
| Search | `page.Search` · `page.SearchRects` · `page.SearchWith` (clip, cap) |
| Outline / links | `doc.TOC`/`SetTOC` · `page.Links`/`InsertLink`/`DeleteLink` |
| Annotate | `page.AddHighlight`/`AddFreeText`/`AddLine`/`Annotations`/`DeleteAnnotation` |
| Redact | `page.AddRedaction` → `page.ApplyRedactions` |
| Forms | `page.Widgets` · `page.SetTextField` · `page.AddTextField` |
| Page ops | `doc.CopyPage`/`MovePage`/`SelectPages`/`InsertPDFRange` |
| Draw | `page.DrawRect`/`DrawLine`/`DrawCircle` · `page.InsertTextbox` |
| Render | `page.Pixmap(PixmapOptions{DPI,CMYK,Clip…})` → `pm.PNG`/`JPEG`/`Save`/`Image` |
| Pixmap ops | `pm.Invert` · `pm.Gamma` · `pm.Bytes` |
| Fonts | `page.GetFonts` · `doc.ExtractFont` · `NewFont(…).TextLength` |
| Images / drawings | `page.GetImages` · `page.ExtractImage` · `page.GetDrawings` |
| Rotation / boxes | `page.Rotation`/`SetRotation` · `page.MediaBox`/`CropBox`/`Set…` |
| Metadata / XMP | `doc.Metadata`/`SetMetadata` · `doc.XMP`/`SetXMP`/`DeleteXMP` |
| Page labels | `doc.PageLabels`/`SetPageLabels` · `page.Label` |
| Build / edit | `NewPDF` · `doc.NewPage` · `doc.DeletePage` · `doc.InsertPDF` |
| Write content | `page.InsertText` · `page.InsertImage` · `page.AddRectAnnot` |
| Save | `doc.SaveBytes` · `doc.EzSave` · `doc.SaveWithOptions` · `doc.SaveIncremental` |

Full reference: **[pkg.go.dev/github.com/srijanmukherjee/gomupdf](https://pkg.go.dev/github.com/srijanmukherjee/gomupdf)**

## Feature support

Legend: ✅ supported · 🟡 partial · ⬜ not yet

| Area | Capability | Status |
|---|---|---|
| Read | Open PDF (file / stream / password) | ✅ |
| Read | Open XPS / EPUB / CBZ / SVG / images, convert to PDF | ✅ |
| Read | Page count, load, iterate | ✅ |
| Read | Plain & structured text, char-level word boxes | ✅ |
| Read | dict/rawdict (per-char), HTML / XHTML / XML export | ✅ |
| Read | Text extraction options (clip rect, stext flags) | ✅ |
| Read | Geometry primitives (`geometry` pkg) | ✅ |
| Read | Text search (quads & rects; region clip + hit cap) | ✅ |
| Read | Tables — word-alignment & vector-ruling, → Markdown | ✅ |
| Structure | Outline (TOC) read & write | ✅ |
| Structure | Page links — read, insert, goto, delete | ✅ |
| Structure | Page labels, page boxes, rotation | ✅ |
| Structure | Metadata read/write + XMP | ✅ |
| Structure | Embedded files, optional-content layers | ⬜ |
| Render | Pixmap → raster (DPI, RGB/Gray/CMYK, clip, annot toggle) | ✅ |
| Render | Pixmap ops (invert, gamma) + PNG/JPEG/PNM encode & save | ✅ |
| Render | Extract images + placement bbox | ✅ |
| Render | Vector drawings / paths | ✅ |
| Read | Fonts — enumerate, extract embedded program, glyph metrics | ✅ |
| Render | OCR, image masks | ⬜ |
| Write | New PDF, page insert/delete/copy/move/select, merge & range-merge | ✅ |
| Write | Insert text, images, word-wrapped text boxes | ✅ |
| Write | Draw lines / rects / circles | ✅ |
| Write | Annotations — markup, shapes, ink, free-text, sticky notes | ✅ |
| Write | Redaction (mark + apply) | ✅ |
| Write | Form widgets — read, fill, create text fields | 🟡 text/checkbox |
| Write | Save options (deflate, clean, linear, ASCII, incremental) | ✅ |
| Write | AES-256 encryption | ✅ |
| Write | Digital signatures, full Shape API, OCG layers | ⬜ |

> gomupdf is actively working toward feature parity with PyMuPDF. The remaining
> remaining ⬜ items (digital signatures, full form types, OCR, optional-content layers) are on the
> roadmap.

## Concurrency & memory

A `Document` owns a single native MuPDF context and is **not** safe for
concurrent use; all of its methods are serialized internally by a mutex. To
process documents in parallel, **open one `Document` per goroutine**.

- Always `Close()` a document when done — the finalizer is only a backstop.
- The in-memory bytes backing a streamed/opened document are held for its
  lifetime (MuPDF reads from them lazily).
- C resources are released deterministically in helper functions, so a Go-side
  mistake cannot leak the underlying objects.

## Backends

gomupdf's public API is **backend-neutral**: every operation that touches a
native engine is delegated through an internal interface. The **MuPDF (cgo)**
backend is the default and is compiled in automatically — you don't need to do
anything.

Building with `-tags nomupdf` excludes all cgo (no C toolchain or MuPDF needed
to compile); constructors then return a "no backend" error. This build tag is
the seam for alternative backends — e.g. a future cgo-free WASM or pure-Go
engine slots in under the opposite tag with no change to the public API.

```sh
go build ./...                 # default: MuPDF backend
go build -tags nomupdf ./...   # compile with the engine excluded
```

## Subpackages

- **[`geometry`](./geometry)** — the 2D primitives (`Point`, `Rect`, `IRect`,
  `Matrix`, `Quad`) used throughout the API, with full affine algebra.
- **[`experimental`](./experimental)** — a higher-level, ergonomic layer:
  lifecycle-managed documents, a row/word model for layout-aware extraction,
  region and key/value queries (`ValueRightOf`, `ValueBelow`), located regex
  search, and helpers for rendering, image extraction, and merging PDFs/images.

## Acknowledgements

- **[MuPDF](https://mupdf.com)** by [Artifex Software](https://artifex.com) is
  the C library that does the heavy lifting — parsing, rendering, and text
  extraction. gomupdf is a thin Go layer over it; all credit for the underlying
  PDF engine belongs to the MuPDF authors.
- **[PyMuPDF](https://github.com/pymupdf/PyMuPDF)** and
  **[pdfplumber](https://github.com/jsvine/pdfplumber)** are the prior art that
  shaped this library's extraction and table-detection design. Thank you to
  their authors and maintainers.

## License

MuPDF is distributed under the **GNU AGPL-3.0**, and this package links against
it, so the same license applies (see [LICENSE](./LICENSE)). Review its
obligations before distributing binaries built with this package.
