package gomupdf

import (
	"errors"
	"strconv"
	"strings"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// Annotation describes an annotation present on a page.
type Annotation struct {
	Index    int           // 0-based position on the page (use with DeleteAnnotation)
	Type     string        // MuPDF annotation type name, e.g. "Highlight", "Square", "FreeText"
	Rect     geometry.Rect // bounding rect in PDF points
	Contents string        // text contents / note, if any
}

// Annotations returns the annotations on the page in document order.
func (p *Page) Annotations() ([]Annotation, error) {
	d := p.doc
	d.mu.Lock()
	if d.b == nil {
		d.mu.Unlock()
		return nil, errors.New("gomupdf: document closed")
	}
	raw, err := d.b.annotationsRaw(p.Number)
	d.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var out []Annotation
	for i, ln := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if ln == "" {
			continue
		}
		// Format: Type\tx0\ty0\tx1\ty1\tContents
		parts := strings.SplitN(ln, "\t", 6)
		if len(parts) < 5 {
			continue
		}
		pf := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
		contents := ""
		if len(parts) == 6 {
			contents = parts[5]
		}
		out = append(out, Annotation{
			Index:    i,
			Type:     parts[0],
			Rect:     geometry.Rect{X0: pf(parts[1]), Y0: pf(parts[2]), X1: pf(parts[3]), Y1: pf(parts[4])},
			Contents: contents,
		})
	}
	return out, nil
}

// DeleteAnnotation removes the annotation at the given 0-based index.
func (p *Page) DeleteAnnotation(index int) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.deleteAnnotation(p.Number, index)
}

// addMarkupAnnot is the shared implementation for highlight/underline/strikeout/squiggly.
func (p *Page) addMarkupAnnot(kind string, quads []geometry.Quad) error {
	if len(quads) == 0 {
		return errors.New("gomupdf: quads must not be empty")
	}
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	// Build flat float array: 8 floats per quad.
	flat := make([]float32, len(quads)*8)
	for i, q := range quads {
		base := i * 8
		flat[base+0] = float32(q.UL.X)
		flat[base+1] = float32(q.UL.Y)
		flat[base+2] = float32(q.UR.X)
		flat[base+3] = float32(q.UR.Y)
		flat[base+4] = float32(q.LL.X)
		flat[base+5] = float32(q.LL.Y)
		flat[base+6] = float32(q.LR.X)
		flat[base+7] = float32(q.LR.Y)
	}
	return d.b.addMarkup(p.Number, kind, flat)
}

// AddHighlight adds a highlight annotation covering the given quads.
// Color: yellow. Changes take effect on the next Save.
func (p *Page) AddHighlight(quads []geometry.Quad) error { return p.addMarkupAnnot("highlight", quads) }

// AddUnderline adds an underline annotation covering the given quads.
// Color: red. Changes take effect on the next Save.
func (p *Page) AddUnderline(quads []geometry.Quad) error { return p.addMarkupAnnot("underline", quads) }

// AddStrikeout adds a strikeout annotation covering the given quads.
// Color: red. Changes take effect on the next Save.
func (p *Page) AddStrikeout(quads []geometry.Quad) error { return p.addMarkupAnnot("strikeout", quads) }

// AddSquiggly adds a squiggly annotation covering the given quads.
// Color: red. Changes take effect on the next Save.
func (p *Page) AddSquiggly(quads []geometry.Quad) error { return p.addMarkupAnnot("squiggly", quads) }
