package gomupdf

import (
	"strconv"
	"strings"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// PathItem is one segment of a drawing path:
//
//	Op "l"  → line:  Pts[0]→Pts[1]
//	Op "c"  → bezier: Pts[0] ctrl1, Pts[1] ctrl2, Pts[2] end (relative to prior point)
//	Op "m"  → moveto: Pts[0]
//	Op "h"  → closepath (no points)
type PathItem struct {
	Op  string
	Pts []geometry.Point
}

// Drawing is one vector path captured from the page.
// Type is "f" (fill) or "s" (stroke). MuPDF reports fill and stroke of the same
// path as separate device calls, so each path's fill and stroke are reported as
// separate entries rather than a single combined entry.
type Drawing struct {
	Type     string
	Color    [3]float64 // sRGB 0..1
	Width    float64    // stroke width, ctm-scaled (0 for fills)
	LineJoin int
	Alpha    float64
	Items    []PathItem
	Rect     geometry.Rect // bounding box of all points
}

// GetDrawings returns the page's vector drawings.
func (p *Page) GetDrawings() ([]Drawing, error) {
	raw, err := p.drawingsRaw()
	if err != nil {
		return nil, err
	}
	var out []Drawing
	var cur *Drawing
	var lastPt geometry.Point
	haveBBox := false
	var bbox geometry.Rect
	add := func(pt geometry.Point) {
		if !haveBBox {
			bbox = geometry.Rect{X0: pt.X, Y0: pt.Y, X1: pt.X, Y1: pt.Y}
			haveBBox = true
			return
		}
		if pt.X < bbox.X0 {
			bbox.X0 = pt.X
		}
		if pt.Y < bbox.Y0 {
			bbox.Y0 = pt.Y
		}
		if pt.X > bbox.X1 {
			bbox.X1 = pt.X
		}
		if pt.Y > bbox.Y1 {
			bbox.Y1 = pt.Y
		}
	}
	flush := func() {
		if cur != nil {
			cur.Rect = bbox
			out = append(out, *cur)
		}
		cur = nil
		haveBBox = false
		bbox = geometry.Rect{}
	}

	for _, ln := range strings.Split(raw, "\n") {
		if ln == "" {
			continue
		}
		f := strings.Fields(ln)
		switch f[0] {
		case "P":
			flush()
			if len(f) < 8 {
				cur = &Drawing{Type: "?"}
				continue
			}
			d := Drawing{Type: f[1]}
			d.Color[0], _ = strconv.ParseFloat(f[2], 64)
			d.Color[1], _ = strconv.ParseFloat(f[3], 64)
			d.Color[2], _ = strconv.ParseFloat(f[4], 64)
			d.Width, _ = strconv.ParseFloat(f[5], 64)
			d.LineJoin, _ = strconv.Atoi(f[6])
			d.Alpha, _ = strconv.ParseFloat(f[7], 64)
			cur = &d
		case "m", "l":
			if cur == nil || len(f) < 3 {
				continue
			}
			x, _ := strconv.ParseFloat(f[1], 64)
			y, _ := strconv.ParseFloat(f[2], 64)
			pt := geometry.Point{X: x, Y: y}
			if f[0] == "l" {
				cur.Items = append(cur.Items, PathItem{Op: "l", Pts: []geometry.Point{lastPt, pt}})
			} else {
				cur.Items = append(cur.Items, PathItem{Op: "m", Pts: []geometry.Point{pt}})
			}
			add(pt)
			lastPt = pt
		case "c":
			if cur == nil || len(f) < 7 {
				continue
			}
			vals := make([]float64, 6)
			for i := 0; i < 6; i++ {
				vals[i], _ = strconv.ParseFloat(f[i+1], 64)
			}
			p1 := geometry.Point{X: vals[0], Y: vals[1]}
			p2 := geometry.Point{X: vals[2], Y: vals[3]}
			p3 := geometry.Point{X: vals[4], Y: vals[5]}
			cur.Items = append(cur.Items, PathItem{Op: "c", Pts: []geometry.Point{p1, p2, p3}})
			add(p1)
			add(p2)
			add(p3)
			lastPt = p3
		case "h":
			if cur != nil {
				cur.Items = append(cur.Items, PathItem{Op: "h"})
			}
		}
	}
	flush()
	return out, nil
}
