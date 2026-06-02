# gomupdf/experimental

A **question-first** layer over `gomupdf`.

The core library hands you a bag of positioned fragments — spans, words, quads,
blocks, bounding boxes — and leaves the geometry to you. That translation
(*question → hand-rolled geometry → primitives → y-flip → tolerance fudge →
answer*) is paid on **every** extraction. This package lets you ask the page
questions instead.

> **Status: experimental.** Heuristics are best-effort and tuned for
> structured, layout-rich documents. Signatures may change.

## Two models, one coordinate system

Everything uses **one** `Rect` (origin top-left, y grows down, units = PDF
points). Conversions from the core library's two rect shapes
(`gomupdf.Rect{X,Y,W,H}` and `geometry.Rect{X0,Y0,X1,Y1}`) live inside this
package so you never juggle both.

### 1. The row/word model — "rows of words"

The substrate most document code actually thinks in. Cluster words into visual
lines, sort by x, filter by x-band, cluster x-positions into columns.

```go
doc, _ := experimental.Open("document.pdf")
defer doc.Close()
page, _ := doc.Page(0)

words, _ := page.Words()                 // []Word with unified Rect + Top/Bottom/Height
rows, _  := page.Rows()                  // visual lines, y-clustered, x-sorted
for _, r := range rows {
    fmt.Println(r.Text())                // words joined left-to-right
}

col   := rows[3].Band(120, 180)          // x-band filter (one column)
lanes := experimental.ClusterFloats(words.Lefts(), 8) // detect column lanes
clean := words.DropOutliers(2.2)         // strip oversized outliers (e.g. watermarks)
```

### 2. The spatial-query model — "ask the page"

Key/value, region cropping, located regex — built on the row/word substrate.

```go
order, ok, _ := page.ValueRightOf("Order No")        // value beside a label
addr,  ok, _ := page.ValueBelow("Address")           // value under a heading
header       := page.TextIn(experimental.R(0, 0, 600, 120)) // crop a region
hits, _      := page.Find(`\d{4}-\d{2}-\d{2}`)       // regex WITH locations
for _, h := range hits {
    fmt.Println(h.Text, h.Rect, h.Context()) // text, box, and the line it sat on
}
```

`CollectBlock` captures the gnarliest reused geometry — walking rows downward
inside an x-band until a stop keyword or a gap (multi-line postal addresses,
label-anchored blocks):

```go
lines := experimental.CollectBlock(rows, labelRow+1, experimental.BlockOptions{
    Lo: 200, Hi: 410, Stop: stopKeywords, GuardLeft: true,
})
```

## Use cases, ranked

| # | Question you ask | API |
|---|------------------|-----|
| 1 | value for this label? | `page.ValueRightOf` / `ValueBelow` |
| 2 | text inside this region? | `page.TextIn(rect)` / `WordsIn(rect)` |
| 3 | where does this pattern appear? | `page.Find(regex)` → located `[]Match` |
| 4 | lines / columns, in order, with boxes? | `page.Rows()`, `Row.Band`, `ClusterFloats` |
| 5 | the table rows? (+ region-scoped) | `page.Tables()` (auto), `TablesIn(rect)` |
| 6 | render / thumbnail? | `page.PNG/Image/SavePNG`, `doc.SavePNGs/Thumbnail` |
| 7 | embedded images? | `page.Images()`, `doc.Images/SaveImages` |
| 8 | quick facts (pages, title, encrypted)? | `doc.Info()`, `doc.Outline()`, `page.Links()` |
| 9 | merge / combine? | `experimental.Merge`, `doc.AppendPDF/AppendImage` |

## Reading & opening

`Open` accepts a **file path, `[]byte`, or `io.Reader`**, with transparent
encryption handling:

```go
doc, err := experimental.Open(src, experimental.Password("1234"))
```

`doc.Raw()` and `page.Raw()` expose the underlying `gomupdf` types as escape
hatches for anything this layer does not wrap.

## Tables

Auto-strategy: tries word-alignment, falls back to vector ruling when alignment
finds nothing — so you need not know the layout up front.

```go
tables, _ := doc.Tables()              // every page, auto, page-tagged
t := tables[0]
fmt.Println(t.NumRows(), t.NumCols(), t.Rows, t.Region())
inBox, _ := page.TablesIn(box, experimental.TableLines()) // force lines, region-scoped
```

## Rendering

```go
png, _   := page.PNG(experimental.DPI(150))
paths, _ := doc.SavePNGs("out/", experimental.DPI(200))   // page-1.png, page-2.png, ...
thumb, _ := doc.Thumbnail(experimental.Zoom(0.3))
```

## Merging (PDFs + images)

```go
experimental.Merge("combined.pdf", "scan.pdf", "receipt.jpg", "appendix.pdf")

doc, _ := experimental.NewDoc()
defer doc.Close()
doc.AppendPDF("a.pdf")
doc.AppendImage("photo.png")   // becomes one full-bleed page sized to the image
doc.Save("out.pdf")
```

## Design notes

- **`Word` edges** map directly onto the names layout code uses: `Left`/`Top`/
  `Right`/`Bottom`, with `Height` (`bottom - top`) doubling as a font-size proxy
  for outlier filtering.
- **Row clustering** sweeps words top-to-bottom and starts a new line when the
  vertical gap exceeds `RowTolerance` (default 3.5 pt) — robust where native
  line breaks are unreliable across columns.
- **`ClusterFloats`** is the shared 1-D clustering primitive behind column-lane
  and row detection.
