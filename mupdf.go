// Package gomupdf is a focused binding for reading, extracting, rendering, and
// writing PDFs. It exposes a compact API: open a (possibly encrypted) PDF from a
// file or from memory, authenticate, and pull reading-order text and positioned
// geometry per page, along with rendering and document-assembly helpers.
//
// The public types here (Document, Page, Font and the result types) are
// backend-neutral; every operation that touches a native engine is delegated to
// the interfaces in backend.go. The default engine is MuPDF (cgo), compiled in
// unless the binary is built with `-tags nomupdf`.
//
// Concurrency: a Document binds one backend and is guarded by a mutex; do not
// share a Document across goroutines without it. Always Close a Document.
package gomupdf

import (
	"errors"
	"iter"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// ---------------------------------------------------------------------------
// Document and page API.
//
// A Document is opened from a file (Open) or from in-memory bytes (OpenStream),
// optionally unlocked with Authenticate, and queried via NeedsPass, PageCount,
// LoadPage, and Pages. Each Page exposes text and geometry extraction. Always
// Close a Document when done.
// ---------------------------------------------------------------------------

// Document is an open PDF. Methods are serialized internally; a Document binds
// one backend. Always Close it.
type Document struct {
	mu        sync.Mutex
	b         docBackend
	locked    bool // encrypted and not yet authenticated
	encrypted bool // whether the PDF was password-protected at open
}

// Open opens a PDF document from a file path.
func Open(filename string) (*Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return OpenStream(data)
}

// OpenStream opens a PDF from in-memory bytes. It does
// not fail on encryption; the returned Document is left locked (NeedsPass()
// true) until Authenticate succeeds.
func OpenStream(pdf []byte) (*Document, error) {
	if len(pdf) == 0 {
		return nil, errors.New("gomupdf: empty input")
	}
	if defaultDriver == nil {
		return nil, errNoBackend
	}
	b, needsPass, err := defaultDriver.open(pdf, ".pdf")
	if err != nil {
		return nil, err
	}
	d := &Document{b: b, locked: needsPass, encrypted: needsPass}
	runtime.SetFinalizer(d, (*Document).Close)
	return d, nil
}

// OpenWithPassword opens a file and authenticates in one step, returning an
// error if the password is wrong (QoL over Open + Authenticate).
func OpenWithPassword(filename, password string) (*Document, error) {
	d, err := Open(filename)
	if err != nil {
		return nil, err
	}
	if d.NeedsPass() && !d.Authenticate(password) {
		d.Close()
		return nil, errors.New("gomupdf: incorrect password")
	}
	return d, nil
}

// OpenStreamWithPassword is OpenWithPassword for in-memory bytes.
func OpenStreamWithPassword(pdf []byte, password string) (*Document, error) {
	d, err := OpenStream(pdf)
	if err != nil {
		return nil, err
	}
	if d.NeedsPass() && !d.Authenticate(password) {
		d.Close()
		return nil, errors.New("gomupdf: incorrect password")
	}
	return d, nil
}

// IsEncrypted reports whether the PDF was password-protected (stays true even
// after successful authentication, unlike NeedsPass).
func (d *Document) IsEncrypted() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.encrypted
}

// NeedsPass reports whether the document is encrypted and not yet unlocked.
// Unlike MuPDF's static flag, this flips to false once Authenticate succeeds.
func (d *Document) NeedsPass() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.locked
}

// Authenticate tries a password and returns true if the document is now
// readable (correct password, or it was never locked).
func (d *Document) Authenticate(password string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return false
	}
	ok := d.b.authenticate(password)
	if ok {
		d.locked = false
	}
	return ok
}

// PageCount returns the number of pages. Returns 0 if the
// document is closed or unreadable.
func (d *Document) PageCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return 0
	}
	return d.b.pageCount()
}

// LoadPage returns a handle to page i (0-based).
// The page is lightweight (it defers to the document on each call); no separate
// Close is required.
func (d *Document) LoadPage(i int) (*Page, error) {
	if i < 0 || i >= d.PageCount() {
		return nil, errors.New("gomupdf: page out of range")
	}
	return &Page{doc: d, Number: i}, nil
}

// Pages iterates pages in order. Use as:
//
//	for i, page := range doc.Pages() { ... }
func (d *Document) Pages() iter.Seq2[int, *Page] {
	return func(yield func(int, *Page) bool) {
		n := d.PageCount()
		for i := 0; i < n; i++ {
			if !yield(i, &Page{doc: d, Number: i}) {
				return
			}
		}
	}
}

// rawText pulls page i's reading-order text (lines '\n'-joined).
func (d *Document) rawText(i int) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.textRaw(i)
}

// Close releases all native resources. Safe to call twice.
func (d *Document) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b != nil {
		d.b.close()
		d.b = nil
	}
	runtime.SetFinalizer(d, nil)
}

// Page is a handle to a single page.
type Page struct {
	doc    *Document
	Number int // 0-based page index
}

// GetText returns the page's reading-order plain text.
func (p *Page) GetText() (string, error) {
	return p.doc.rawText(p.Number)
}

// Bound returns the page's bounding box in points.
func (p *Page) Bound() (geometry.Rect, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return geometry.Rect{}, errors.New("gomupdf: document closed")
	}
	raw, err := d.b.boundRaw(p.Number)
	if err != nil {
		return geometry.Rect{}, err
	}
	f := strings.Fields(raw)
	if len(f) != 4 {
		return geometry.Rect{}, errors.New("gomupdf: bad bound output")
	}
	x0, _ := strconv.ParseFloat(f[0], 64)
	y0, _ := strconv.ParseFloat(f[1], 64)
	x1, _ := strconv.ParseFloat(f[2], 64)
	y1, _ := strconv.ParseFloat(f[3], 64)
	return geometry.Rect{X0: x0, Y0: y0, X1: x1, Y1: y1}, nil
}

// --- write API (Go entry points; backend does the work) --------------------

// NewPDF creates a new, empty PDF document.
func NewPDF() (*Document, error) {
	if defaultDriver == nil {
		return nil, errNoBackend
	}
	b, err := defaultDriver.newPDF()
	if err != nil {
		return nil, err
	}
	d := &Document{b: b}
	runtime.SetFinalizer(d, (*Document).Close)
	return d, nil
}

// SaveBytes serializes the document to PDF bytes. garbage collects unused objects.
func (d *Document) SaveBytes(garbage bool) ([]byte, error) {
	opts := SaveOptions{}
	if garbage {
		opts.Garbage = 1
	}
	return d.SaveBytesWithOptions(opts)
}

// Save writes the document to a PDF file.
func (d *Document) Save(path string, garbage bool) error {
	data, err := d.SaveBytes(garbage)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// SaveEncryptedBytes serializes the document with AES-256 encryption. userPwd
// is required to open; ownerPwd grants full permissions (may be "").
func (d *Document) SaveEncryptedBytes(userPwd, ownerPwd string) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	return d.b.saveEncrypted(userPwd, ownerPwd)
}

// NewPage appends a blank page of the given size (points).
func (d *Document) NewPage(width, height float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.newPage(width, height)
}

// DeletePage removes page n (0-based).
func (d *Document) DeletePage(n int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.deletePage(n)
}

// InsertPDF appends every page of the source PDF (given as bytes) to this
// document. password unlocks an encrypted source ("" if none).
func (d *Document) InsertPDF(src []byte, password string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	if len(src) == 0 {
		return errors.New("gomupdf: empty source")
	}
	return d.b.insertPDF(src, password)
}

// InsertText draws a line of Helvetica text at (x, y) (baseline) on page i.
func (p *Page) InsertText(x, y, size float64, text string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.insertText(p.Number, x, y, size, text)
}

// InsertImage places an encoded image (jpeg/png/…) into rect r on page i.
func (p *Page) InsertImage(r geometry.Rect, img []byte) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	if len(img) == 0 {
		return errors.New("gomupdf: empty image")
	}
	return d.b.insertImage(p.Number, [4]float64{r.X0, r.Y0, r.X1, r.Y1}, img)
}

// AddRectAnnot adds a rectangle (square) annotation on page i.
func (p *Page) AddRectAnnot(r geometry.Rect) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addRectAnnot(p.Number, [4]float64{r.X0, r.Y0, r.X1, r.Y1})
}

// imagesRaw returns the image-metadata trace output (IMG ... lines).
func (p *Page) imagesRaw() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.imagesRaw(p.Number)
}

// imageBytesRaw returns the encoded bytes + extension of the index-th image.
func (p *Page) imageBytesRaw(index int) ([]byte, string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return nil, "", errors.New("gomupdf: document closed")
	}
	return d.b.imageBytes(p.Number, index)
}

// drawingsRaw returns the trace-device output (P/m/l/c/h lines).
func (p *Page) drawingsRaw() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.drawingsRaw(p.Number)
}

// tocRaw returns the outline output ("level\tpage\ttitle" per line).
func (d *Document) tocRaw() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.tocRaw()
}

// linksRaw returns the link output ("x0 y0 x1 y1\turi" per line).
func (p *Page) linksRaw() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.linksRaw(p.Number)
}

// searchRaw returns the search output (one hit per line: 8 quad floats).
func (p *Page) searchRaw(needle string) (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.searchRaw(p.Number, needle)
}

// wordsRaw returns the word-extraction output (one word per line:
// "x0 y0 x1 y1 block line\t<text>"). Parsed by Page.Words.
func (p *Page) wordsRaw() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.wordsRaw(p.Number)
}

// StructuredJSON returns structured text as JSON (blocks → lines → chars, each
// with bbox). This is the raw geometry feed for layout reconstruction; see
// Words/Blocks for typed access.
func (p *Page) StructuredJSON() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.structuredJSON(p.Number)
}

const softHyphen = "­" // U+00AD

// Lines returns the page's non-empty lines with the soft hyphen (U+00AD)
// stripped and trailing whitespace trimmed.
func (p *Page) Lines() ([]string, error) {
	raw, err := p.GetText()
	if err != nil {
		return nil, err
	}
	return splitLines(raw), nil
}

func splitLines(raw string) []string {
	raw = strings.ReplaceAll(raw, softHyphen, "")
	var out []string
	for _, ln := range strings.Split(raw, "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		out = append(out, strings.TrimRight(ln, " \t\r"))
	}
	return out
}

// AllLines returns every page's Lines concatenated into a single flat line
// stream across the whole document.
func (d *Document) AllLines() ([]string, error) {
	var out []string
	for _, page := range d.Pages() {
		lines, err := page.Lines()
		if err != nil {
			return nil, err
		}
		out = append(out, lines...)
	}
	return out, nil
}

// Text returns the full document text, pages separated by a form feed ("\f").
func (d *Document) Text() (string, error) {
	var b strings.Builder
	for i, page := range d.Pages() {
		if i > 0 {
			b.WriteByte('\f')
		}
		t, err := page.GetText()
		if err != nil {
			return "", err
		}
		b.WriteString(t)
	}
	return b.String(), nil
}

// TextByPage returns each page's text as a slice.
func (d *Document) TextByPage() ([]string, error) {
	var out []string
	for _, page := range d.Pages() {
		t, err := page.GetText()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// metaKeys are the standard metadata lookups exposed by Metadata.
var metaKeys = map[string]string{
	"format":       "format",
	"encryption":   "encryption",
	"title":        "info:Title",
	"author":       "info:Author",
	"subject":      "info:Subject",
	"keywords":     "info:Keywords",
	"creator":      "info:Creator",
	"producer":     "info:Producer",
	"creationDate": "info:CreationDate",
	"modDate":      "info:ModDate",
}

// Metadata returns document metadata (title, author, format, …). Absent keys are
// omitted.
func (d *Document) Metadata() (map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	out := make(map[string]string)
	for name, key := range metaKeys {
		v, ok := d.b.lookupMeta(key)
		if !ok {
			continue
		}
		if v = strings.TrimSpace(v); v != "" {
			out[name] = v
		}
	}
	return out, nil
}
