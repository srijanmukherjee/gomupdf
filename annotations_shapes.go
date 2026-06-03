package gomupdf

import (
	"errors"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// AnnotStyle holds shared styling options for shape and text annotations.
// The zero value uses sensible defaults: black 1pt stroke, no fill, fully opaque.
type AnnotStyle struct {
	Stroke  *[3]float64 // RGB 0..1 border/line color; nil → black
	Fill    *[3]float64 // RGB 0..1 interior fill color; nil → no fill
	Width   float64     // border width in points; ≤0 → 1
	Opacity float64     // 0..1; ≤0 → 1 (fully opaque)
}

// pointsToFlat32 converts a slice of geometry.Point into a flat []float32 [x0,y0,x1,y1,...].
func pointsToFlat32(pts []geometry.Point) []float32 {
	flat := make([]float32, len(pts)*2)
	for i, p := range pts {
		flat[i*2] = float32(p.X)
		flat[i*2+1] = float32(p.Y)
	}
	return flat
}

// AddLine adds a Line annotation from a to b on the page.
func (p *Page) AddLine(a, b geometry.Point, style AnnotStyle) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addLineAnnot(p.Number, [2]float64{a.X, a.Y}, [2]float64{b.X, b.Y}, style)
}

// AddCircle adds a Circle (ellipse) annotation inscribed in rect r.
func (p *Page) AddCircle(r geometry.Rect, style AnnotStyle) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addCircleAnnot(p.Number, [4]float64{r.X0, r.Y0, r.X1, r.Y1}, style)
}

// AddPolygon adds a closed Polygon annotation with the given vertices.
func (p *Page) AddPolygon(pts []geometry.Point, style AnnotStyle) error {
	if len(pts) < 2 {
		return errors.New("gomupdf: polygon requires at least 2 points")
	}
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addPolyAnnot(p.Number, true, pointsToFlat32(pts), style)
}

// AddPolyline adds an open PolyLine annotation with the given vertices.
func (p *Page) AddPolyline(pts []geometry.Point, style AnnotStyle) error {
	if len(pts) < 2 {
		return errors.New("gomupdf: polyline requires at least 2 points")
	}
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addPolyAnnot(p.Number, false, pointsToFlat32(pts), style)
}

// AddInk adds a freehand Ink annotation with one or more strokes.
func (p *Page) AddInk(strokes [][]geometry.Point, style AnnotStyle) error {
	if len(strokes) == 0 {
		return errors.New("gomupdf: ink requires at least one stroke")
	}
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	// Flatten all strokes into one point array and build per-stroke counts.
	counts := make([]int32, len(strokes))
	var allPts []geometry.Point
	for i, s := range strokes {
		counts[i] = int32(len(s))
		allPts = append(allPts, s...)
	}
	return d.b.addInkAnnot(p.Number, counts, pointsToFlat32(allPts), style)
}

// AddFreeText adds a FreeText annotation with the given text and font size.
func (p *Page) AddFreeText(r geometry.Rect, text string, size float64, style AnnotStyle) error {
	if size <= 0 {
		size = 12
	}
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addFreeText(p.Number, [4]float64{r.X0, r.Y0, r.X1, r.Y1}, text, size, style)
}

// AddTextNote adds a Text (sticky note) annotation at the given point.
func (p *Page) AddTextNote(at geometry.Point, text string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addTextNote(p.Number, [2]float64{at.X, at.Y}, text)
}
