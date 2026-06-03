package gomupdf

// Backend abstraction
// ===================
//
// gomupdf's public API (Document, Page, Font and the result types) is
// backend-neutral: every operation that actually touches a native engine goes
// through one of the interfaces below. The MuPDF/cgo implementation lives in
// the same package but is guarded by the `!nomupdf` build tag and registers
// itself as the default driver via init(). Building with `-tags nomupdf`
// excludes all cgo; constructors then return errNoBackend until an alternative
// backend (e.g. a future pure-Go or WASM implementation) is compiled in under
// the opposite tag.
//
// The interfaces are deliberately "narrow and primitive": methods exchange
// strings, bytes, and scalars (plus the package's plain option structs), and
// all parsing into the public result types stays in the backend-neutral layer.
// This keeps a backend small to implement and keeps result types in one place.
//
// These types are unexported: they are an internal SPI, not part of the public
// API. Page-level operations take a 0-based page number rather than a separate
// page handle, mirroring the underlying engine calls.

import "errors"

// errNoBackend is returned by constructors when the binary was built without a
// backend (e.g. `-tags nomupdf`) and no driver registered itself.
var errNoBackend = errors.New("gomupdf: no backend compiled in (build without the 'nomupdf' tag, or supply an alternative backend)")

// defaultDriver is set by a backend's init() (the MuPDF backend does this in a
// `!nomupdf`-tagged file). nil means no backend is available.
var defaultDriver driver

// concurrentBackend is an optional capability: a docBackend that can spawn
// read-only worker clones for parallel render/extract. Each clone shares the
// base document's caches but carries independent per-goroutine engine state, so
// multiple clones may run read methods concurrently without serialization. A
// backend that does not implement this (or a closed document) makes the
// concurrency helpers in concurrent.go fall back to serial execution.
type concurrentBackend interface {
	// cloneWorker returns a read-only backend sharing this document's content.
	// The clone must be closed by the caller and must not outlive the base.
	cloneWorker() (docBackend, error)
}

// driver opens and creates documents and fonts for one engine.
type driver interface {
	name() string
	// open parses an in-memory document. magic is a format hint (a filename,
	// extension, or mime type; "" or ".pdf" means PDF). needsPass reports
	// whether the opened document is password-locked.
	open(data []byte, magic string) (doc docBackend, needsPass bool, err error)
	// newPDF creates a new, empty PDF document.
	newPDF() (docBackend, error)
	// newFont loads one of the 14 standard PDF fonts by name.
	newFont(name string) (fontBackend, error)
}

// docBackend is one open document. Methods are NOT internally synchronized; the
// public Document serializes them under its mutex. Page-level methods take a
// 0-based page number.
type docBackend interface {
	close()
	authenticate(password string) bool
	pageCount() int

	// --- whole-document serialization -------------------------------------
	save(opts SaveOptions, orig []byte) ([]byte, error)
	saveEncrypted(userPwd, ownerPwd string) ([]byte, error)
	convertToPDF() ([]byte, error)

	// --- metadata & structure ---------------------------------------------
	lookupMeta(key string) (value string, present bool)
	setMetadata(spec string) error
	xmp() ([]byte, error)
	setXMP(xml []byte) error
	deleteXMP() error
	tocRaw() (string, error)
	setTOC(spec string) error
	pageLabelsRaw() (string, error)
	setPageLabels(spec string) error
	pageLabel(pageNo int) (string, error)

	// --- page tree editing -------------------------------------------------
	newPage(width, height float64) error
	deletePage(n int) error
	copyPage(from, to int) error
	movePage(from, to int) error
	selectPages(pages []int) error
	insertPDF(src []byte, password string) error
	insertPDFRange(src []byte, fromPage, toPage int, password string) error

	// --- per-page geometry -------------------------------------------------
	boundRaw(pageNo int) (string, error)
	geometryRaw(pageNo int) (string, error)
	setRotation(pageNo, deg int) error
	setBox(pageNo, which int, x0, y0, x1, y1 float64) error

	// --- per-page text extraction -----------------------------------------
	textRaw(pageNo int) (string, error)
	extractTextRaw(pageNo int, opts TextOptions) (string, error)
	wordsRaw(pageNo int) (string, error)
	structuredJSON(pageNo int) (string, error)
	rawDictRaw(pageNo int) (string, error)
	htmlRaw(pageNo int) (string, error)
	xhtmlRaw(pageNo int) (string, error)
	xmlRaw(pageNo int) (string, error)
	searchRaw(pageNo int, needle string) (string, error)

	// --- per-page images, drawings, fonts, links --------------------------
	imagesRaw(pageNo int) (string, error)
	imageBytes(pageNo, index int) (data []byte, ext string, err error)
	drawingsRaw(pageNo int) (string, error)
	fontsRaw(pageNo int) (string, error)
	extractFont(xref int) (name, ext string, data []byte, err error)
	linksRaw(pageNo int) (string, error)
	insertLink(pageNo int, rect [4]float64, uri string) error
	insertGotoLink(pageNo int, rect [4]float64, destPage int) error
	deleteLink(pageNo, index int) error

	// --- per-page rendering ------------------------------------------------
	pixmap(pageNo int, opts PixmapOptions) ([]byte, error)

	// --- per-page content writing -----------------------------------------
	insertText(pageNo int, x, y, size float64, text string) error
	insertImage(pageNo int, rect [4]float64, img []byte) error
	drawContent(pageNo int, fragment string) error
	drawText(pageNo int, fragment string) error

	// --- per-page annotations ---------------------------------------------
	annotationsRaw(pageNo int) (string, error)
	deleteAnnotation(pageNo, index int) error
	addRectAnnot(pageNo int, rect [4]float64) error
	addMarkup(pageNo int, kind string, quads []float32) error
	addLineAnnot(pageNo int, a, b [2]float64, style AnnotStyle) error
	addCircleAnnot(pageNo int, rect [4]float64, style AnnotStyle) error
	addPolyAnnot(pageNo int, closed bool, pts []float32, style AnnotStyle) error
	addInkAnnot(pageNo int, counts []int32, pts []float32, style AnnotStyle) error
	addFreeText(pageNo int, rect [4]float64, text string, size float64, style AnnotStyle) error
	addTextNote(pageNo int, at [2]float64, text string) error

	// --- per-page redaction & widgets -------------------------------------
	addRedaction(pageNo int, rect [4]float64, fill *[3]float64) error
	applyRedactions(pageNo int) (int, error)
	widgetsRaw(pageNo int) (string, error)
	setTextField(pageNo, index int, value string) error
	setCheckbox(pageNo, index int, checked bool) error
	addTextField(pageNo int, name string, rect [4]float64, value string) error
}

// fontBackend is a loaded font for measuring text.
type fontBackend interface {
	close()
	fontName() string
	advance(r rune) float64 // horizontal advance in em units (1.0 = one em)
	ascender() float64
	descender() float64
}
