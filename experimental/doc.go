package experimental

import (
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/srijanmukherjee/gomupdf"
)

// ErrPassword is returned by Open when a document is encrypted and the supplied
// password is missing or incorrect. Test for it with errors.Is.
var ErrPassword = errors.New("experimental: document is encrypted and the password is missing or incorrect")

// Doc is a thin, lifecycle-managed handle over a gomupdf.Document. It exists so
// callers stop juggling Open/Authenticate/Close ceremony and page handles.
//
//	doc, err := experimental.Open("document.pdf", experimental.Password("1234"))
//	if err != nil { ... }
//	defer doc.Close()
//	for _, page := range doc.Pages() { ... }
type Doc struct {
	raw *gomupdf.Document
}

// OpenOption configures Open.
type OpenOption func(*openConfig)

type openConfig struct {
	password string
}

// Password supplies the password for an encrypted PDF.
func Password(pw string) OpenOption {
	return func(c *openConfig) { c.password = pw }
}

// Open opens a PDF from any of: a file path (string), raw bytes ([]byte), or an
// io.Reader (read fully into memory). Encryption is handled transparently when a
// Password option is given; otherwise an encrypted PDF returns an error.
func Open(src any, opts ...OpenOption) (*Doc, error) {
	var c openConfig
	for _, o := range opts {
		o(&c)
	}

	raw, err := openRaw(src)
	if err != nil {
		return nil, err
	}
	// Own authentication here so both the missing-password and wrong-password
	// cases surface as the single ErrPassword sentinel. An unneeded password on
	// an unencrypted document is harmless (NeedsPass is false, so we never auth).
	if raw.NeedsPass() && !raw.Authenticate(c.password) {
		raw.Close()
		return nil, ErrPassword
	}
	return &Doc{raw: raw}, nil
}

func openRaw(src any) (*gomupdf.Document, error) {
	switch v := src.(type) {
	case string:
		return gomupdf.Open(v)
	case []byte:
		return gomupdf.OpenStream(v)
	case io.Reader:
		b, err := io.ReadAll(v)
		if err != nil {
			return nil, fmt.Errorf("experimental: read source: %w", err)
		}
		return gomupdf.OpenStream(b)
	default:
		return nil, fmt.Errorf("experimental: unsupported source type %T (want string, []byte, or io.Reader)", src)
	}
}

// Close releases native resources. Safe to call more than once.
func (d *Doc) Close() {
	if d != nil && d.raw != nil {
		d.raw.Close()
	}
}

// NumPages returns the page count.
func (d *Doc) NumPages() int { return d.raw.PageCount() }

// Raw exposes the underlying gomupdf.Document as an escape hatch for features
// this layer does not wrap (write/modify, metadata, TOC, drawings, ...).
func (d *Doc) Raw() *gomupdf.Document { return d.raw }

// Page returns the 0-based page i as an experimental.Page.
func (d *Doc) Page(i int) (*Page, error) {
	p, err := d.raw.LoadPage(i)
	if err != nil {
		return nil, err
	}
	return &Page{raw: p, idx: i}, nil
}

// Pages iterates pages in order: for i, page := range doc.Pages() { ... }.
// Pages that fail to load are skipped.
func (d *Doc) Pages() iter.Seq2[int, *Page] {
	return func(yield func(int, *Page) bool) {
		for i := 0; i < d.raw.PageCount(); i++ {
			p, err := d.raw.LoadPage(i)
			if err != nil {
				continue
			}
			if !yield(i, &Page{raw: p, idx: i}) {
				return
			}
		}
	}
}

// Text returns the whole document's text, pages separated by form feed.
func (d *Doc) Text() (string, error) { return d.raw.Text() }

// Lines returns every page's cleaned lines (soft hyphens stripped, blanks
// dropped) as one flat stream — a flat line stream suitable for line-oriented
// parsing.
func (d *Doc) Lines() ([]string, error) { return d.raw.AllLines() }
