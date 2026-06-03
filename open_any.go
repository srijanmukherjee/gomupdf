package gomupdf

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Opening non-PDF formats. MuPDF's document handlers cover XPS/OXPS, EPUB,
// MOBI, FB2, CBZ, SVG, plain text, and raster images (PNG, JPEG, GIF, BMP,
// TIFF, …). These open read-only as ordinary Documents — text, geometry, and
// rendering all work — but only true PDFs can be written back; use ConvertToPDF
// to turn any opened document into a writable PDF Document.

// OpenAny opens a document of any MuPDF-supported format, inferring the format
// from the file extension. Use Open for the PDF-only fast path.
func OpenAny(filename string) (*Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	return OpenAnyStream(data, ext)
}

// OpenAnyStream opens in-memory bytes of any MuPDF-supported format. filetype
// is a format hint — a file extension ("png", "xps", "epub"), a leading-dot
// extension (".pdf"), or a MIME type. An empty filetype is treated as PDF.
func OpenAnyStream(data []byte, filetype string) (*Document, error) {
	if len(data) == 0 {
		return nil, errors.New("gomupdf: empty input")
	}
	if defaultDriver == nil {
		return nil, errNoBackend
	}
	if filetype == "" {
		filetype = ".pdf"
	}
	b, needsPass, err := defaultDriver.open(data, filetype)
	if err != nil {
		return nil, err
	}
	d := &Document{b: b, locked: needsPass, encrypted: needsPass}
	runtime.SetFinalizer(d, (*Document).Close)
	return d, nil
}

// ConvertToPDF renders every page of the document into a freshly assembled PDF
// and returns it as a new, writable Document. The receiver is left untouched
// and must still be closed independently. For a document that is already PDF
// this produces a clean structural copy.
func (d *Document) ConvertToPDF() (*Document, error) {
	d.mu.Lock()
	if d.b == nil {
		d.mu.Unlock()
		return nil, errors.New("gomupdf: document closed")
	}
	data, err := d.b.convertToPDF()
	d.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return OpenStream(data)
}
