// Package experimental is a question-first layer over gomupdf.
//
// The base library hands you a bag of positioned fragments (spans, words,
// quads, blocks) and leaves the geometry to you. This package lets you ask
// the page questions instead:
//
//	v, _ := page.ValueRightOf("Total")   // key/value extraction
//	t    := page.TextIn(box)             // crop text by region
//	hits := page.Find(`\d{4}-\d{4}`)     // regex with locations
//
// One coordinate model is used everywhere: Rect, origin top-left, y grows
// down, units are PDF points. Conversions from the base library's two rect
// shapes (gomupdf.Rect{X,Y,W,H} and geometry.Rect{X0,Y0,X1,Y1}) live here so
// callers never juggle both.
//
// Status: experimental. Heuristics are best-effort and tuned for structured,
// layout-rich documents. Signatures may change.
package experimental

import (
	"github.com/srijanmukherjee/gomupdf"
	"github.com/srijanmukherjee/gomupdf/geometry"
)

// Rect is an axis-aligned rectangle in PDF points. Origin is top-left and y
// grows downward, matching the page coordinate system used throughout gomupdf.
type Rect struct {
	X0, Y0, X1, Y1 float64
}

// R is a terse constructor for a Rect from its two corners.
func R(x0, y0, x1, y1 float64) Rect { return Rect{x0, y0, x1, y1} }

func (r Rect) Width() float64   { return r.X1 - r.X0 }
func (r Rect) Height() float64  { return r.Y1 - r.Y0 }
func (r Rect) CenterX() float64 { return (r.X0 + r.X1) / 2 }
func (r Rect) CenterY() float64 { return (r.Y0 + r.Y1) / 2 }

// Empty reports whether the rect has no area.
func (r Rect) Empty() bool { return r.Width() <= 0 || r.Height() <= 0 }

// Contains reports whether s lies entirely within r.
func (r Rect) Contains(s Rect) bool {
	return s.X0 >= r.X0 && s.Y0 >= r.Y0 && s.X1 <= r.X1 && s.Y1 <= r.Y1
}

// ContainsPoint reports whether (x, y) lies within r.
func (r Rect) ContainsPoint(x, y float64) bool {
	return x >= r.X0 && x <= r.X1 && y >= r.Y0 && y <= r.Y1
}

// Overlaps reports whether r and s share any area.
func (r Rect) Overlaps(s Rect) bool {
	return r.X0 < s.X1 && r.X1 > s.X0 && r.Y0 < s.Y1 && r.Y1 > s.Y0
}

// Union returns the smallest rect enclosing both r and s. The zero Rect is
// treated as "nothing", so unioning onto it yields s unchanged.
func (r Rect) Union(s Rect) Rect {
	if (r == Rect{}) {
		return s
	}
	if (s == Rect{}) {
		return r
	}
	return Rect{min(r.X0, s.X0), min(r.Y0, s.Y0), max(r.X1, s.X1), max(r.Y1, s.Y1)}
}

// Expand grows the rect by d points on every side (negative shrinks it).
func (r Rect) Expand(d float64) Rect {
	return Rect{r.X0 - d, r.Y0 - d, r.X1 + d, r.Y1 + d}
}

// Geometry converts back to the base library's corner rect, for callers that
// need to hand a region to gomupdf APIs (e.g. InsertImage, AddRectAnnot).
func (r Rect) Geometry() geometry.Rect { return geometry.NewRect(r.X0, r.Y0, r.X1, r.Y1) }

// rectFromMu converts gomupdf.Rect{X,Y,W,H} (used by Word/Span boxes).
func rectFromMu(m gomupdf.Rect) Rect { return Rect{m.X, m.Y, m.X + m.W, m.Y + m.H} }

// rectFromGeometry converts geometry.Rect{X0,Y0,X1,Y1} (used by Bound/Search).
func rectFromGeometry(g geometry.Rect) Rect { return Rect{g.X0, g.Y0, g.X1, g.Y1} }
