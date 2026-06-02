// Package geometry provides a self-contained 2D geometry toolkit for PDF
// coordinates: Point, Rect, IRect, Matrix, and Quad. Operations that other
// languages might express through operators are provided as explicit methods:
//
//	add          → Add               invert       → Inverted
//	subtract     → Sub               intersect    → Intersect
//	scale        → Mul               contains pt  → ContainsPoint
//	transform    → Transform         contains rect→ ContainsRect
//	as tuple     → Tuple             set element  → Set
package geometry

import (
	"errors"
	"math"
)

// EPSILON is the tolerance used for matrix near-equality comparisons.
const EPSILON = 1e-5

// eq is the tolerance for float equality of components (expected values are
// exact, so a tight tolerance only absorbs floating-point noise).
const eq = 1e-9

// snapEps: angles within this of a quadrant snap to exact 0/±1, so rotations by
// multiples of 90 degrees yield clean values.
const snapEps = 1e-4

// Identity matrix.
var Identity = Matrix{1, 0, 0, 1, 0, 0}

var errIndex = errors.New("geometry: index out of range")

func feq(a, b float64) bool { return math.Abs(a-b) <= eq }

// ---------------------------------------------------------------------------
// Point
// ---------------------------------------------------------------------------

type Point struct{ X, Y float64 }

func NewPoint(x, y float64) Point { return Point{x, y} }

func (p Point) Tuple() [2]float64         { return [2]float64{p.X, p.Y} }
func (p Point) Equal(q Point) bool        { return feq(p.X, q.X) && feq(p.Y, q.Y) }
func (p Point) Add(q Point) Point         { return Point{p.X + q.X, p.Y + q.Y} }
func (p Point) Sub(q Point) Point         { return Point{p.X - q.X, p.Y - q.Y} }
func (p Point) Mul(s float64) Point       { return Point{p.X * s, p.Y * s} }
func (p Point) AddScalar(s float64) Point { return Point{p.X + s, p.Y + s} }

func (p Point) Transform(m Matrix) Point {
	return Point{p.X*m.A + p.Y*m.C + m.E, p.X*m.B + p.Y*m.D + m.F}
}

func (p Point) Unit() Point {
	l := math.Hypot(p.X, p.Y)
	if l == 0 {
		return Point{}
	}
	return Point{p.X / l, p.Y / l}
}

func (p Point) AbsUnit() Point {
	l := math.Hypot(p.X, p.Y)
	if l == 0 {
		return Point{}
	}
	return Point{math.Abs(p.X) / l, math.Abs(p.Y) / l}
}

func (p Point) DistanceToPoint(q Point) float64 { return math.Hypot(p.X-q.X, p.Y-q.Y) }

func (p Point) DistanceToRect(r Rect) float64 {
	r = r.Normalize()
	dx := math.Max(math.Max(r.X0-p.X, 0), p.X-r.X1)
	dy := math.Max(math.Max(r.Y0-p.Y, 0), p.Y-r.Y1)
	return math.Hypot(dx, dy)
}

func (p Point) Get(i int) (float64, error) {
	switch i {
	case 0:
		return p.X, nil
	case 1:
		return p.Y, nil
	}
	return 0, errIndex
}

func (p *Point) Set(i int, v float64) error {
	switch i {
	case 0:
		p.X = v
	case 1:
		p.Y = v
	default:
		return errIndex
	}
	return nil
}

// ---------------------------------------------------------------------------
// Rect (x0,y0,x1,y1)
// ---------------------------------------------------------------------------

type Rect struct{ X0, Y0, X1, Y1 float64 }

func NewRect(x0, y0, x1, y1 float64) Rect { return Rect{x0, y0, x1, y1} }
func RectFromPoints(a, b Point) Rect      { return Rect{a.X, a.Y, b.X, b.Y} }

func (r Rect) Tuple() [4]float64 { return [4]float64{r.X0, r.Y0, r.X1, r.Y1} }

func (r Rect) Equal(s Rect) bool {
	return feq(r.X0, s.X0) && feq(r.Y0, s.Y0) && feq(r.X1, s.X1) && feq(r.Y1, s.Y1)
}

func (r Rect) IsEmpty() bool   { return r.X0 >= r.X1 || r.Y0 >= r.Y1 }
func (r Rect) IsValid() bool   { return r.X0 <= r.X1 && r.Y0 <= r.Y1 }
func (r Rect) Width() float64  { return r.X1 - r.X0 }
func (r Rect) Height() float64 { return r.Y1 - r.Y0 }

func (r Rect) Normalize() Rect {
	if r.X1 < r.X0 {
		r.X0, r.X1 = r.X1, r.X0
	}
	if r.Y1 < r.Y0 {
		r.Y0, r.Y1 = r.Y1, r.Y0
	}
	return r
}

func (r Rect) IncludePoint(p Point) Rect {
	return Rect{math.Min(r.X0, p.X), math.Min(r.Y0, p.Y), math.Max(r.X1, p.X), math.Max(r.Y1, p.Y)}
}

func (r Rect) IncludeRect(s Rect) Rect {
	if s.IsEmpty() || !s.IsValid() {
		return r
	}
	return Rect{math.Min(r.X0, s.X0), math.Min(r.Y0, s.Y0), math.Max(r.X1, s.X1), math.Max(r.Y1, s.Y1)}
}

func (r Rect) Add(s Rect) Rect          { return Rect{r.X0 + s.X0, r.Y0 + s.Y0, r.X1 + s.X1, r.Y1 + s.Y1} }
func (r Rect) Sub(s Rect) Rect          { return Rect{r.X0 - s.X0, r.Y0 - s.Y0, r.X1 - s.X1, r.Y1 - s.Y1} }
func (r Rect) Mul(s float64) Rect       { return Rect{r.X0 * s, r.Y0 * s, r.X1 * s, r.Y1 * s} }
func (r Rect) DivScalar(s float64) Rect { return Rect{r.X0 / s, r.Y0 / s, r.X1 / s, r.Y1 / s} }
func (r Rect) DivMatrix(m Matrix) Rect  { return r.Transform(m.Inverted()) }

// Transform maps the rect's four corners by m and returns their bounding box.
func (r Rect) Transform(m Matrix) Rect {
	pts := []Point{
		(Point{r.X0, r.Y0}).Transform(m),
		(Point{r.X1, r.Y0}).Transform(m),
		(Point{r.X0, r.Y1}).Transform(m),
		(Point{r.X1, r.Y1}).Transform(m),
	}
	out := Rect{pts[0].X, pts[0].Y, pts[0].X, pts[0].Y}
	for _, p := range pts[1:] {
		out.X0 = math.Min(out.X0, p.X)
		out.Y0 = math.Min(out.Y0, p.Y)
		out.X1 = math.Max(out.X1, p.X)
		out.Y1 = math.Max(out.Y1, p.Y)
	}
	return out
}

func (r Rect) Intersect(s Rect) Rect {
	return Rect{math.Max(r.X0, s.X0), math.Max(r.Y0, s.Y0), math.Min(r.X1, s.X1), math.Min(r.Y1, s.Y1)}
}

func (r Rect) Intersects(s Rect) bool {
	i := r.Intersect(s)
	return i.X0 < i.X1 && i.Y0 < i.Y1
}

func (r Rect) ContainsPoint(p Point) bool {
	return p.X >= r.X0 && p.X < r.X1 && p.Y >= r.Y0 && p.Y < r.Y1
}

func (r Rect) ContainsRect(s Rect) bool {
	return s.X0 >= r.X0 && s.X1 <= r.X1 && s.Y0 >= r.Y0 && s.Y1 <= r.Y1
}

func (r Rect) Quad() Quad { return Quad{UL: r.TL(), UR: r.TR(), LL: r.BL(), LR: r.BR()} }

func (r Rect) TL() Point { return Point{r.X0, r.Y0} }
func (r Rect) TR() Point { return Point{r.X1, r.Y0} }
func (r Rect) BL() Point { return Point{r.X0, r.Y1} }
func (r Rect) BR() Point { return Point{r.X1, r.Y1} }

func (r Rect) Get(i int) (float64, error) {
	switch i {
	case 0:
		return r.X0, nil
	case 1:
		return r.Y0, nil
	case 2:
		return r.X1, nil
	case 3:
		return r.Y1, nil
	}
	return 0, errIndex
}

func (r *Rect) Set(i int, v float64) error {
	switch i {
	case 0:
		r.X0 = v
	case 1:
		r.Y0 = v
	case 2:
		r.X1 = v
	case 3:
		r.Y1 = v
	default:
		return errIndex
	}
	return nil
}

// ---------------------------------------------------------------------------
// IRect (integer rect)
// ---------------------------------------------------------------------------

type IRect struct{ X0, Y0, X1, Y1 int }

func NewIRect(x0, y0, x1, y1 int) IRect { return IRect{x0, y0, x1, y1} }
func IRectFromPoints(a, b Point) IRect  { return IRect{int(a.X), int(a.Y), int(b.X), int(b.Y)} }

func (r IRect) Tuple() [4]int      { return [4]int{r.X0, r.Y0, r.X1, r.Y1} }
func (r IRect) Equal(s IRect) bool { return r == s }
func (r IRect) Rect() Rect {
	return Rect{float64(r.X0), float64(r.Y0), float64(r.X1), float64(r.Y1)}
}

func (r IRect) IncludePoint(p Point) IRect {
	return IRect{imin(r.X0, int(p.X)), imin(r.Y0, int(p.Y)), imax(r.X1, int(p.X)), imax(r.Y1, int(p.Y))}
}

func (r IRect) IncludeRect(s IRect) IRect {
	if s.X0 >= s.X1 || s.Y0 >= s.Y1 { // empty
		return r
	}
	return IRect{imin(r.X0, s.X0), imin(r.Y0, s.Y0), imax(r.X1, s.X1), imax(r.Y1, s.Y1)}
}

func (r IRect) Get(i int) (int, error) {
	switch i {
	case 0:
		return r.X0, nil
	case 1:
		return r.Y0, nil
	case 2:
		return r.X1, nil
	case 3:
		return r.Y1, nil
	}
	return 0, errIndex
}

func (r *IRect) Set(i int, v int) error {
	switch i {
	case 0:
		r.X0 = v
	case 1:
		r.Y0 = v
	case 2:
		r.X1 = v
	case 3:
		r.Y1 = v
	default:
		return errIndex
	}
	return nil
}

// ---------------------------------------------------------------------------
// Matrix (a,b,c,d,e,f)
// ---------------------------------------------------------------------------

type Matrix struct{ A, B, C, D, E, F float64 }

func NewMatrix(a, b, c, d, e, f float64) Matrix { return Matrix{a, b, c, d, e, f} }

func Rotate(deg float64) Matrix {
	s, c := sinCosSnap(deg)
	return Matrix{c, s, -s, c, 0, 0}
}
func Scale(sx, sy float64) Matrix { return Matrix{sx, 0, 0, sy, 0, 0} }
func Shear(h, v float64) Matrix   { return Matrix{1, v, h, 1, 0, 0} }

// sinCosSnap returns sin,cos of deg, snapping to exact values near quadrants.
func sinCosSnap(deg float64) (sin, cos float64) {
	for deg < 0 {
		deg += 360
	}
	for deg >= 360 {
		deg -= 360
	}
	switch {
	case math.Abs(deg-0) < snapEps:
		return 0, 1
	case math.Abs(deg-90) < snapEps:
		return 1, 0
	case math.Abs(deg-180) < snapEps:
		return 0, -1
	case math.Abs(deg-270) < snapEps:
		return -1, 0
	}
	rad := deg * math.Pi / 180
	return math.Sin(rad), math.Cos(rad)
}

func (m Matrix) Tuple() [6]float64 { return [6]float64{m.A, m.B, m.C, m.D, m.E, m.F} }

func (m Matrix) Equal(n Matrix) bool {
	return feq(m.A, n.A) && feq(m.B, n.B) && feq(m.C, n.C) && feq(m.D, n.D) && feq(m.E, n.E) && feq(m.F, n.F)
}

func (m Matrix) Abs() float64 {
	return math.Sqrt(m.A*m.A + m.B*m.B + m.C*m.C + m.D*m.D + m.E*m.E + m.F*m.F)
}

func (m Matrix) Add(n Matrix) Matrix {
	return Matrix{m.A + n.A, m.B + n.B, m.C + n.C, m.D + n.D, m.E + n.E, m.F + n.F}
}
func (m Matrix) Sub(n Matrix) Matrix {
	return Matrix{m.A - n.A, m.B - n.B, m.C - n.C, m.D - n.D, m.E - n.E, m.F - n.F}
}
func (m Matrix) AddScalar(s float64) Matrix {
	return Matrix{m.A + s, m.B + s, m.C + s, m.D + s, m.E + s, m.F + s}
}
func (m Matrix) MulScalar(s float64) Matrix {
	return Matrix{m.A * s, m.B * s, m.C * s, m.D * s, m.E * s, m.F * s}
}
func (m Matrix) DivScalar(s float64) Matrix {
	return Matrix{m.A / s, m.B / s, m.C / s, m.D / s, m.E / s, m.F / s}
}

// Mul concatenates m with n (m * n), matching fz_concat.
func (m Matrix) Mul(n Matrix) Matrix {
	return Matrix{
		A: m.A*n.A + m.B*n.C,
		B: m.A*n.B + m.B*n.D,
		C: m.C*n.A + m.D*n.C,
		D: m.C*n.B + m.D*n.D,
		E: m.E*n.A + m.F*n.C + n.E,
		F: m.E*n.B + m.F*n.D + n.F,
	}
}

// Concat returns a * b (receiver ignored).
func (m Matrix) Concat(a, b Matrix) Matrix { return a.Mul(b) }

func (m Matrix) Inverted() Matrix {
	det := m.A*m.D - m.B*m.C
	if math.Abs(det) < 1e-12 {
		return Matrix{} // non-invertible → zero
	}
	out := Matrix{A: m.D / det, B: -m.B / det, C: -m.C / det, D: m.A / det}
	out.E = -m.E*out.A - m.F*out.C
	out.F = -m.E*out.B - m.F*out.D
	return out
}

func (m *Matrix) Invert() bool {
	det := m.A*m.D - m.B*m.C
	if math.Abs(det) < 1e-12 {
		return false
	}
	*m = m.Inverted()
	return true
}

func (m Matrix) Pretranslate(x, y float64) Matrix {
	m.E += x*m.A + y*m.C
	m.F += x*m.B + y*m.D
	return m
}

func (m Matrix) Prescale(sx, sy float64) Matrix {
	m.A *= sx
	m.B *= sx
	m.C *= sy
	m.D *= sy
	return m
}

func (m Matrix) Preshear(h, v float64) Matrix {
	a, b := m.A, m.B
	m.A += v * m.C
	m.B += v * m.D
	m.C += h * a
	m.D += h * b
	return m
}

func (m Matrix) Prerotate(deg float64) Matrix {
	s, c := sinCosSnap(deg)
	a, b, cc, d := m.A, m.B, m.C, m.D
	m.A = c*a + s*cc
	m.B = c*b + s*d
	m.C = -s*a + c*cc
	m.D = -s*b + c*d
	return m
}

func (m Matrix) Get(i int) (float64, error) {
	switch i {
	case 0:
		return m.A, nil
	case 1:
		return m.B, nil
	case 2:
		return m.C, nil
	case 3:
		return m.D, nil
	case 4:
		return m.E, nil
	case 5:
		return m.F, nil
	}
	return 0, errIndex
}

func (m *Matrix) Set(i int, v float64) error {
	switch i {
	case 0:
		m.A = v
	case 1:
		m.B = v
	case 2:
		m.C = v
	case 3:
		m.D = v
	case 4:
		m.E = v
	case 5:
		m.F = v
	default:
		return errIndex
	}
	return nil
}

// ---------------------------------------------------------------------------
// Quad (four corners)
// ---------------------------------------------------------------------------

type Quad struct{ UL, UR, LL, LR Point }

// polygon returns the corners in traversal order UL→UR→LR→LL.
func (q Quad) polygon() [4]Point { return [4]Point{q.UL, q.UR, q.LR, q.LL} }

func (q Quad) IsRectangular() bool {
	return feq(q.UL.Y, q.UR.Y) && feq(q.LL.Y, q.LR.Y) &&
		feq(q.UL.X, q.LL.X) && feq(q.UR.X, q.LR.X)
}

func (q Quad) IsEmpty() bool {
	pts := q.polygon()
	area := 0.0
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		area += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	return math.Abs(area/2) < eq
}

func (q Quad) IsConvex() bool {
	pts := q.polygon()
	var sign float64
	for i := 0; i < 4; i++ {
		a := pts[i]
		b := pts[(i+1)%4]
		c := pts[(i+2)%4]
		cross := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
		if math.Abs(cross) < eq {
			continue
		}
		if sign == 0 {
			sign = cross
		} else if (cross > 0) != (sign > 0) {
			return false
		}
	}
	return true
}

func (q Quad) Rect() Rect {
	pts := q.polygon()
	out := Rect{pts[0].X, pts[0].Y, pts[0].X, pts[0].Y}
	for _, p := range pts[1:] {
		out.X0 = math.Min(out.X0, p.X)
		out.Y0 = math.Min(out.Y0, p.Y)
		out.X1 = math.Max(out.X1, p.X)
		out.Y1 = math.Max(out.Y1, p.Y)
	}
	return out
}

func (q Quad) Transform(m Matrix) Quad {
	return Quad{UL: q.UL.Transform(m), UR: q.UR.Transform(m), LL: q.LL.Transform(m), LR: q.LR.Transform(m)}
}

func (q Quad) ContainsPoint(p Point) bool {
	pts := q.polygon()
	var sign float64
	for i := 0; i < 4; i++ {
		a := pts[i]
		b := pts[(i+1)%4]
		cross := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
		if math.Abs(cross) < eq {
			continue // on edge
		}
		if sign == 0 {
			sign = cross
		} else if (cross > 0) != (sign > 0) {
			return false
		}
	}
	return true
}

func (q Quad) ContainsRect(r Rect) bool {
	for _, p := range []Point{r.TL(), r.TR(), r.BR(), r.BL()} {
		if !q.ContainsPoint(p) {
			return false
		}
	}
	return true
}

func (q Quad) ContainsQuad(o Quad) bool {
	for _, p := range []Point{o.UL, o.UR, o.LR, o.LL} {
		if !q.ContainsPoint(p) {
			return false
		}
	}
	return true
}

func (q Quad) Get(i int) (Point, error) {
	switch i {
	case 0:
		return q.UL, nil
	case 1:
		return q.UR, nil
	case 2:
		return q.LL, nil
	case 3:
		return q.LR, nil
	}
	return Point{}, errIndex
}

func (q *Quad) Set(i int, p Point) error {
	switch i {
	case 0:
		q.UL = p
	case 1:
		q.UR = p
	case 2:
		q.LL = p
	case 3:
		q.LR = p
	default:
		return errIndex
	}
	return nil
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
