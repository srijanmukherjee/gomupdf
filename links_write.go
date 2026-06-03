package gomupdf

import (
	"errors"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// InsertLink adds a clickable link over rect that opens an external URI.
// Changes take effect on the next Save.
func (p *Page) InsertLink(rect geometry.Rect, uri string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.insertLink(p.Number, [4]float64{rect.X0, rect.Y0, rect.X1, rect.Y1}, uri)
}

// InsertGotoLink adds an internal link over rect that jumps to destPage (0-based).
// Changes take effect on the next Save.
func (p *Page) InsertGotoLink(rect geometry.Rect, destPage int) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.insertGotoLink(p.Number, [4]float64{rect.X0, rect.Y0, rect.X1, rect.Y1}, destPage)
}

// DeleteLink removes the link at the given 0-based index (in Links() order).
// Changes take effect on the next Save.
func (p *Page) DeleteLink(index int) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.deleteLink(p.Number, index)
}
