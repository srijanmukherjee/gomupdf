package gomupdf

import (
	"errors"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// RedactOptions controls how covered content is removed.
type RedactOptions struct {
	// Fill is the RGB fill (0..1 each) drawn over the redacted area.
	// nil defaults to black {0,0,0}.
	Fill *[3]float64

	// RemoveImages removes images that overlap the redaction region when true.
	RemoveImages bool
}

// AddRedaction marks rect for redaction (a redaction annotation). Nothing is
// removed until ApplyRedactions is called.
func (p *Page) AddRedaction(rect geometry.Rect, opts RedactOptions) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addRedaction(p.Number, [4]float64{rect.X0, rect.Y0, rect.X1, rect.Y1}, opts.Fill)
}

// ApplyRedactions permanently removes the content under every redaction
// annotation on the page (text, and images per options), then deletes the
// redaction marks. Returns the number of redactions applied. Effective on Save.
func (p *Page) ApplyRedactions() (int, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return 0, errors.New("gomupdf: document closed")
	}
	return d.b.applyRedactions(p.Number)
}
