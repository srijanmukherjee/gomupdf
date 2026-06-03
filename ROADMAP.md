# gomupdf Roadmap

Goal: reach **PyMuPDF (`fitz`) feature parity at v1.0 in one month**, then spend the
following year building the things PyMuPDF structurally *can't* or *won't* do — turning
gomupdf from "a Go binding" into the best PDF library in the Go ecosystem.

> **Reality check on the one-month v1.** Full literal parity with PyMuPDF (Story/HTML
> layout, digital signatures, XFA, journaling/undo, OCR) is not buildable in four weeks
> by one person — and PyMuPDF itself ships several of those as separate packages or
> thin/version-dependent APIs. So **v1.0 = parity on the 80% surface that real users
> actually touch** (open/save, full text extraction, tables, rendering, images,
> drawings, annotations, links, forms-read, basic editing, redaction). The hard 20%
> (signatures, OCR, Story, layers) is explicitly scheduled post-v1 as the year's
> differentiators. This is the honest scope; everything below is planned against it.

---

## North star — ship killer workflows, not a feature checklist

The biggest unlock is **not** "support every PDF feature." PyMuPDF exploded because it aligned
with real workflows. So that's the lens: feature parity is the *means*; a small set of
**workflow-complete recipes** is the *end*. A capability isn't "done" when the API exists — it's
done when the end-to-end workflow it serves is documented, has a runnable example, and is
benchmarked. We pick a few workflows to be *the best in the Go ecosystem* at, and let the rest follow.

| Workflow | Powered by (primitives) | Status / lands in |
|---|---|---|
| **Render pages to images** (thumbnails, previews) | `Pixmap`/`Render` (DPI, colorspace, clip), PNG/JPEG encode | ✅ shipping (v0.8) |
| **Table extraction → Markdown** | `FindTables`, `Table.ToMarkdown` | ✅ shipping (v0.7) |
| **"Ask the page" data extraction** (invoices, statements) | `ValueRightOf`/`ValueBelow`, `TextIn`, regex `Find`, → struct unmarshal | 🟡 experimental → promote |
| **PDF → Markdown** (clean, structured) | reading-order, headings, tables→GFM, image refs | `v1.4` |
| **Document chunking for RAG** | per-page chunks + metadata, token-aware splits | `v1.4` |
| **Blazing-fast PDF merging** | `InsertPDF`/`Merge` + **concurrency** (fan-out) | `v1.2` (concurrency) |
| **Document/invoice stamping** (watermarks, Bates) | `InsertText`/`InsertImage`/overlay + Shape API | W4 + `v1.x` recipe |
| **OCR extraction** (scanned docs) | auto-detect + Tesseract + pluggable engines | `v1.5` |

Each milestone below should leave at least one of these workflows **fully shippable** — recipe in
the README, example in `examples/`, and a benchmark vs PyMuPDF/pdfium where relevant. Parity features
that don't serve a target workflow are lower priority than polishing the workflows that do.

---

## Where we are today (baseline)

**Have (stable root pkg):** open file/stream/password, `NewPDF`, AES-256 save, `Save`/`SaveBytes`
with garbage GC, page count, page iterate/load, `Bound`, reading-order text, `Words`/`Spans`/
`StructuredText`/`StructuredJSON`, `FindTables` (text + lines strategies), `GetImages`/`ExtractImage`,
`GetDrawings`, `Pixmap` (zoom/gray/alpha) + PNG, `Search` (quads/rects), `TOC` (read), `Links`
(read, URI only), full geometry pkg (Point/Rect/IRect/Matrix/Quad), and editing primitives
(`NewPage`, `DeletePage`, `InsertPDF`, `InsertText` Helvetica-only, `InsertImage`, `AddRectAnnot`).

**Have (experimental pkg):** question-first query API — `Find` (regex), `Search`, `ValueRightOf`/
`ValueBelow`, `Rows`/`ClusterRows`, `TextIn`, region tables, render convenience (`Image`/`PNG`/
`SavePNGs`), unified `Rect`, lifecycle `Doc`, `Merge`.

**The headline gaps vs PyMuPDF:** metadata r/w, page rotation/boxes, `get_text` dict/rawdict/html/xml
+ clip + flags, table→markdown + merged cells, full Pixmap ops + colorspaces, full drawings detail,
the entire annotation write surface, link write + non-URI kinds, forms/widgets, TOC write, the rich
editing/Shape API, redaction, fonts, embedded files, layers, non-PDF formats, OCR, signatures, Story.

---

## Month 1 — the v1.0 parity sprint

Four weekly milestones. Each ships as a tagged `v0.x` pre-release; week 4 close = `v1.0.0`.
Order is leverage-first and dependency-aware (text engine before things that build on it).

### Week 1 — `v0.6` Document & page core
- **Metadata** read/write (`Metadata()`, `SetMetadata`), XMP get/set/del.
- **Page geometry**: `Rotation`/`SetRotation`, `MediaBox`, `CropBox`, rotation matrices.
- **Save options**: `deflate`, `deflate_images/fonts`, `clean`, `linear`, `pretty`, `ascii`,
  incremental save (`SaveIncremental`), `ez_save`-style defaults, permissions bitmask.
- **Open any format**: route `fz_open_document` so XPS/OXPS/EPUB/MOBI/FB2/CBZ/SVG/TXT/images open;
  `ConvertToPDF()`. (MuPDF does the heavy lifting — mostly surface + filetype hinting.)
- **Page labels** read/write (cheap, lives here).
- io.Reader/io.Writer + `context.Context`-aware open/save (Go-native ergonomics, not in PyMuPDF).

### Week 2 — `v0.7` The text & table engine (highest value)
- `get_text` full mode set: **`dict`/`rawdict`** (blocks→lines→spans→**per-char** bbox/origin/color/
  flags), **`blocks`** tuple form, **`html`/`xhtml`/`xml`**.
- **`clip` rect** + **flags** on every text call (dehyphenate, preserve-whitespace/ligatures/images,
  inhibit-spaces, mediabox-clip) + **`sort`** reading-order toggle.
- **`TextPage` reuse** — build once, extract many ways (perf + parity).
- **Reading-order reconstruction** as a first-class mode (multi-column aware) — a recurring PyMuPDF
  pain, we make it default-good.
- **Tables**: `ToMarkdown()` (GitHub-flavored), header detection, **merged-cell** row/col spans,
  per-cell bbox, links/images inside cells, and **combined text+tables in reading order**
  (`get_text_and_tables` equivalent — PyMuPDF discussion #3095).
- **Search**: flags + quads-by-default for highlight placement.

### Week 3 — `v0.8` Rendering, images, drawings, fonts
- **Pixmap** to parity: CMYK + colorspace control, `clip` rect, `annots` toggle, multi-format encode
  (`tobytes`/save: png/jpg/pnm/psd/ps), `set_alpha`, `invert_irect`, `gamma_with`, `tint_with`,
  `shrink`/`scale`, `copy`, pixel get/set, `set_dpi`, `color_count`.
- **Images**: `get_image_info` (placement/transform/bbox for displayed + inline images),
  `get_image_rects`/`get_image_bbox`, and **composited** "image as rendered" (masks/SMasks applied).
- **Drawings** to parity: all item types (line/rect/quad/bezier/curve), fill+stroke+`fs`, dashes,
  line cap/join, `even_odd`, opacity, closePath; `get_cdrawings` fast path; `cluster_drawings`.
- **Fonts**: `get_fonts`, `extract_font`, `Font` object — `text_length`, glyph advance/bbox,
  `has_glyph`, ascender/descender (needed for correct text insertion + redaction height).

### Week 4 — `v0.9 → v1.0.0` Annotations, links, forms, editing, redaction
- **Annotations** write surface: highlight/underline/strikeout/squiggly, text (sticky), freetext,
  line, rect, circle, polygon, polyline, ink, stamp, file-attachment; `update()` appearance, colors/
  border/opacity/flags/info, iterate, delete.
- **Links**: full `get_links` (goto/gotor/uri/launch/named), `insert/update/delete_link`, `resolve_link`.
- **TOC write**: `set_toc`, `set_toc_item`, outline edit.
- **Editing**: `insert_textbox` (alignment/fit), `insert_image` options (rotate/mask/keep-prop),
  `insert_font`, draw primitives (line/rect/circle/oval/polyline/bezier/curve/quad), the **Shape API**
  (batched vector state), `copy_page`/`move_page`/`select`/`fullcopy_page`, `insert_pdf` with page range.
- **Redaction**: `add_redact_annot` + `apply_redactions` with image/graphics/text granularity flags.
- **Forms (read + basic fill)**: enumerate `Widget`s (text/checkbox/radio/listbox/combobox/signature),
  read field name/type/value/flags, set text/checkbox values + `update()`.
- Docs, examples, `pkg.go.dev` polish, benchmark suite → **tag `v1.0.0`**.

> **Explicitly deferred past v1.0** (and why): Story/HTML layout + DocumentWriter (large, PyMuPDF
> ships it semi-separately), OCR (large, external Tesseract), digital-signature *creation* (crypto +
> incremental-save correctness), XFA dynamic forms (niche/large), journaling/undo, full OCG/layers,
> embedded-file API. All scheduled below.

---

## The year — post-v1: build what PyMuPDF can't

Two strategic bets define the product, both playing to Go's strengths and PyMuPDF's structural gaps:
**(1) real concurrency**, and **(2) in-core LLM/Markdown export** — plus a structural enabler that
comes first: **a backend abstraction** decoupling gomupdf's public API from MuPDF. Build it
immediately after v1.0, because it is the foundation for thread-safety, license freedom, and cgo
removal (see [Architecture evolution](#architecture-evolution-the-long-game)).

### Foundation · Month 2 — `v1.1` Backend abstraction

- Introduce an internal **`backend` interface** behind the public types (`Document`, `Page`,
  `geometry`, `Word`, `Table`, `Pixmap` — already largely engine-neutral). MuPDF becomes one
  implementation under `backend/mupdf`; **no public API change**.
- This is the hinge for the rest of the year: the concurrency model, alternative engines (permissive
  or commercial — the AGPL escape), and a cgo-free WASM engine all become *backend choices* rather
  than rewrites. Ship it before the differentiators so they're built once, against the interface.

### Q1 remainder · Months 2–3 — `v1.2 → v1.4` Differentiators + hardening

- **`v1.2` Concurrency (the killer feature) — ✅ delivered.** PyMuPDF flatly does not support threads
  (issues #107, #1994, #4760) and tells users to multiprocess. We ship a **goroutine-safe model**: a
  render/extract **worker pool** (`MapPages`, `RenderPages`, `TextByPageConcurrent`) with
  `context.Context` cancellation, results returned in page order. "Fan a large PDF across all your
  cores" is one API call (~4.7× on the heavy render benchmark, 8 cores). Design note: rather than the
  clone-context-over-shared-document approach originally sketched below, each worker gets a **fully
  independent context *and* document over the shared immutable input bytes** — MuPDF's lazy page-tree
  parsing mutates a shared document's object cache outside the store locks, so concurrent page loads on
  one document corrupt it (confirmed under `-race`). Independent documents share no mutable engine
  state, cover extraction as well as rendering, and keep the single-threaded path lock-free. Encrypted
  and in-memory (`NewPDF`) documents transparently fall back to serial. Guarded by `-race` in CI.
- **`v1.3` Memory stability.** PyMuPDF has a long leak tail in loops (render #1430, insert-image #714,
  insert_pdf #1738, save #2791, merge #3201). We commit to **leak-free batch workloads**: rigorous
  `fz_drop_*` discipline, finalizers as backstop, `Close()`/`defer` everywhere, ASAN/valgrind in CI,
  long-running soak test + fuzzing on malformed PDFs (#3344).
- **`v1.4` Markdown / LLM export in-core.** PyMuPDF makes you reach for the separate `pymupdf4llm`.
  We ship **`ToMarkdown()` / RAG chunking** built in: multi-column reading order, font-size→heading
  mapping, tables→GFM, inline bold/italic, image refs, per-page chunks with metadata. This is the
  single hottest use case in the ecosystem — own it.

### Q2 · Months 4–6 — `v1.5 → v1.7` Text quality & scale

- **`v1.5` OCR.** Tesseract via MuPDF, but with the ergonomics PyMuPDF lacks: **auto-detect** which
  pages need OCR (hybrid, ~2× faster), clean engine config (no `TESSDATA_PREFIX` dance, #3374),
  alpha/mask handling fixed (#3842), and a **pluggable OCR interface** so RapidOCR/cloud engines drop in.
- **`v1.6` International text.** **Bidi/RTL** (Arabic/Hebrew returned in correct order + ligature
  normalization, #2199) and **CJK font fallback bundled by default** (no `pymupdf-fonts` install gotcha).
- **`v1.7` Scale & precision.** **Streaming/lazy** large-PDF API with bounded memory; a **fast
  plain-text path** to compete with pdfium on the simple case; **precise redaction** (tight,
  configurable bbox overlap — fixes the "redacts adjacent text" complaints #696/#2762/#526/#434).

### Q3 · Months 7–9 — `v1.8 → v1.10` Documents & forms

- **`v1.8` Containers**: embedded files/attachments API, **OCG/optional-content layers** (add/toggle/
  bind), page-label edge cases.
- **`v1.9` Form filling completeness**: interconnected radio groups, correct **appearance-stream
  generation** so filled forms render in Chrome/all viewers (NeedAppearances, #2789/#563), arbitrary
  field fonts (#1697).
- **`v1.10` Story/HTML layout**: flowable HTML/CSS engine + `DocumentWriter` (paginated output),
  `insert_htmlbox`, `Archive` resource bundles.

### Q4 · Months 10–12 — `v2.0` Trust, polish, stability

- **Digital signatures**: PKCS#12 **signing** (not just reading) + verification, with correct
  **incremental save** — a hard PyMuPDF "no," high value for contract/compliance workflows.
- **Document hygiene**: `scrub` (strip metadata/JS/links), `bake` (flatten annots/widgets),
  `subset_fonts`, `recolor`.
- **Distribution**: prebuilt cgo artifacts for **musl + glibc (incl. old glibc / AWS Lambda)** so
  `go get` "just works" — Go's structural advantage over PyMuPDF's wheel pain (#1955).
- **Stabilize**: promote the `experimental` query API into a supported package, freeze the **v2 API**,
  full docs + cookbook + benchmarks vs PyMuPDF/pdfium, clear **licensing** story (AGPL inheritance from
  MuPDF + commercial path) → **tag `v2.0.0`**.

---

## New capabilities beyond parity (the gomupdf identity)

Things that make gomupdf *better*, not just equal:

1. **Goroutine-safe concurrent rendering/extraction** — Go's home turf; PyMuPDF can't.
2. **In-core LLM/RAG export** — `ToMarkdown`, chunking + metadata, no bolt-on package.
3. **Question-first query API** — already prototyped in `experimental`: `ValueRightOf`/`ValueBelow`,
   region crops, regex find. Extend to **schema/struct extraction** — "unmarshal a PDF into a Go struct"
   via field tags (invoice/statement parsing in a few lines).
4. **First-class reading-order** + bidi/CJK correct by default.
5. **Streaming + bounded-memory** large-PDF processing.
6. **Native Go interfaces** — `io.Reader`/`io.Writer`, `context.Context` cancellation, `iter.Seq2`
   page iteration (already present), errors via `errors.Is`.
7. **"Just works" install** — prebuilt binaries across libc variants.
8. **Precise, auditable redaction** — security-grade, configurable.

---

## Architecture evolution (the long game)

Three structural goals that outlive the feature roadmap. All three are unlocked by the **`v1.1`
backend abstraction** — once the public API no longer names MuPDF, thread-safety becomes a backend
property, license freedom becomes a backend choice, and cgo removal becomes a backend implementation.

### A. Thread safety — ✅ shipped in `v1.2`

Each `Document` still owns one `fz_context` guarded by a `sync.Mutex`; its own methods serialize
(MuPDF rule #2 — `fz_context` is not thread-safe). The original plan was MuPDF's clone-context
protocol: install `fz_locks_context` callbacks, `fz_clone_context` per goroutine, and load pages from
those clones over a **shared document**. We prototyped exactly that and it **corrupted under `-race`**:
MuPDF resolves PDF objects/page-tree nodes lazily, mutating the *document's* object cache — and the
store locks do not cover that cache, so concurrent page loads on one document race. (MuPDF's own
example dodges this by loading all pages on the main thread and only rasterizing in parallel, which
would exclude parallel text extraction.)

**What shipped instead — independent state per worker:** each worker gets its own context **and** its
own document opened over the same **immutable input buffer** (`cloneWorker` in `mupdf_backend.go`).
Workers share no mutable MuPDF state, so render *and* extract parallelize safely; contexts stay
`NULL`-locked so the single-threaded path pays no mutex overhead. The Go side is a bounded worker pool
(`MapPages`/`RenderPages`/`TextByPageConcurrent` in `concurrent.go`) with `context.Context`
cancellation and in-order results; encrypted/in-memory documents fall back to serial. Tested under
`-race` in CI (ASAN/TSAN + long soak tests remain future hardening).

The single biggest differentiator vs PyMuPDF, which refuses threading.

### B. License freedom — removing MuPDF / escaping AGPL

MuPDF (and therefore this binding) is **AGPL-3.0**. Two separable goals:

- **Escape AGPL, keep MuPDF (cheap, near-term):** buy a **commercial license from Artifex** (MuPDF is
  dual-licensed) — zero code change. Or stay AGPL and target OSS/server-with-source users.
- **Actually remove MuPDF (large, multi-year):** behind the backend interface, add a
  **permissively-licensed engine** — *pdfium* (BSD-3; excellent render/text, heavy build, would
  require rebuilding MuPDF's stext/table/drawings extraction) or *pdfcpu* (Apache-2.0, pure Go; strong
  structure/write but **no rasterization**, weak layout-aware text). Default to it once it reaches
  parity; MuPDF becomes optional/commercial.

Honest take: MuPDF's extraction is best-in-class; full replacement at today's feature richness is a
multi-year effort. Near-term answer = **commercial license + backend interface so users can swap**.

### C. Removing the cgo dependency

cgo removal means no C toolchain at build and clean cross-compilation. Three paths, recommended order:

1. **WASM + wazero (credible near-term).** Compile MuPDF (or pdfium) to `wasm32-wasi`, `go:embed` the
   module, drive it with the pure-Go [wazero](https://wazero.io) runtime. Removes the C toolchain and
   delivers the "`go build` just works" distribution win. Cost: ~1.5–3× slower, boundary memory copies,
   a wasm build pipeline; still AGPL if it's MuPDF.
2. **Pure-Go stack (largest).** pdfcpu (parse/write) + a Go rasterizer (`x/image/vector`, `fogleman/gg`)
   + Go text shaping (`go-text/typesetting`) + a content-stream interpreter — effectively a PDF render
   engine in Go. Realistic only for a *subset* (fast plain-text + structure) while heavy rendering
   stays on the WASM/native engine.
3. **C→Go transpilation** (the `modernc.org/sqlite` technique). MuPDF is far larger and more complex
   than SQLite; very high effort, fragile. Not recommended.

Recommended sequence: backend interface → **wazero+WASM backend** (cgo-free, cross-compilable, keeps
the engine) → optional pure-Go backend for the no-C-engine subset later.

## Release cadence & versioning

- **Weekly** tagged pre-releases during the v1 sprint (`v0.6`–`v0.9`), then `v1.0.0`.
- **Monthly** minor releases post-v1 (`v1.1`–`v1.10` → `v2.0`), patch as needed.
- SemVer; experimental pkg stays `v0`-style unstable until promoted at v2.
- Every release: CHANGELOG entry, migration notes on breaks, `pkg.go.dev` docs, benchmark deltas.
