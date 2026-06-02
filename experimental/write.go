package experimental

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"os"

	// Register decoders so AppendImage/Merge can read image dimensions.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/srijanmukherjee/gomupdf"
	"github.com/srijanmukherjee/gomupdf/geometry"
)

// NewDoc creates a new, empty PDF document ready for AppendPDF / AppendImage.
func NewDoc() (*Doc, error) {
	raw, err := gomupdf.NewPDF()
	if err != nil {
		return nil, err
	}
	return &Doc{raw: raw}, nil
}

// AppendPDF appends every page of a source PDF (path, []byte, or io.Reader).
func (d *Doc) AppendPDF(src any) error {
	b, err := resolveBytes(src)
	if err != nil {
		return err
	}
	return d.raw.InsertPDF(b, "")
}

// AppendImage appends the image (path, []byte, or io.Reader) as a new full-bleed
// page sized to the image's pixel dimensions (1px = 1pt). Encoded JPEG/PNG/GIF
// bytes are accepted as-is.
func (d *Doc) AppendImage(src any) error {
	b, err := resolveBytes(src)
	if err != nil {
		return err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("experimental: decode image: %w", err)
	}
	w, h := float64(cfg.Width), float64(cfg.Height)
	if err := d.raw.NewPage(w, h); err != nil {
		return err
	}
	p, err := d.raw.LoadPage(d.raw.PageCount() - 1)
	if err != nil {
		return err
	}
	return p.InsertImage(geometry.NewRect(0, 0, w, h), b)
}

// Bytes serializes the document to PDF bytes (with garbage collection).
func (d *Doc) Bytes() ([]byte, error) { return d.raw.SaveBytes(true) }

// Save writes the document to a PDF file (with garbage collection).
func (d *Doc) Save(path string) error { return d.raw.Save(path, true) }

// Merge stitches any mix of PDFs and images into a single PDF at out. Each
// source (path, []byte, or io.Reader) is sniffed: PDFs have all their pages
// appended; images each become one page.
//
//	experimental.Merge("combined.pdf", "scan.pdf", "receipt.jpg", "appendix.pdf")
func Merge(out string, sources ...any) error {
	d, err := NewDoc()
	if err != nil {
		return err
	}
	defer d.Close()
	for i, src := range sources {
		b, err := resolveBytes(src)
		if err != nil {
			return fmt.Errorf("source %d: %w", i, err)
		}
		if isPDF(b) {
			if err := d.raw.InsertPDF(b, ""); err != nil {
				return fmt.Errorf("source %d (pdf): %w", i, err)
			}
			continue
		}
		if err := d.AppendImage(b); err != nil {
			return fmt.Errorf("source %d (image): %w", i, err)
		}
	}
	return d.Save(out)
}

func isPDF(b []byte) bool { return bytes.HasPrefix(bytes.TrimLeft(b, " \r\n\t"), []byte("%PDF")) }

// resolveBytes reads a source given as a file path (string), raw bytes
// ([]byte), or an io.Reader into a byte slice.
func resolveBytes(src any) ([]byte, error) {
	switch v := src.(type) {
	case string:
		return os.ReadFile(v)
	case []byte:
		return v, nil
	case io.Reader:
		return io.ReadAll(v)
	default:
		return nil, fmt.Errorf("experimental: unsupported source type %T (want string, []byte, or io.Reader)", src)
	}
}
