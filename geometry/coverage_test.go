package geometry

import (
	"math"
	"testing"
)

// These tests fill the remaining coverage gaps in geometry.go: zero-coverage
// accessors (Width/Height, Rect.ContainsRect, the Get methods, IRect.Rect,
// Quad.Rect/Set), the out-of-range error branches of every Get/Set, and the
// uncovered conditional branches in Normalize, Unit, AbsUnit, Invert,
// sinCosSnap, IsConvex, ContainsPoint/ContainsRect/ContainsQuad, and imin/imax.

func TestPointGetSet(t *testing.T) {
	p := NewPoint(7, 9)
	cases := []struct {
		i    int
		want float64
	}{{0, 7}, {1, 9}}
	for _, c := range cases {
		got, err := p.Get(c.i)
		if err != nil {
			t.Fatalf("Point.Get(%d) err = %v", c.i, err)
		}
		if got != c.want {
			t.Errorf("Point.Get(%d) = %v, want %v", c.i, got, c.want)
		}
	}
	for _, bad := range []int{-1, 2, 99} {
		if _, err := p.Get(bad); err == nil {
			t.Errorf("Point.Get(%d) expected error", bad)
		}
	}
	// Set valid then bad.
	var q Point
	if err := q.Set(0, 3); err != nil || q.X != 3 {
		t.Errorf("Point.Set(0) = %v err %v", q.X, err)
	}
	if err := q.Set(1, 4); err != nil || q.Y != 4 {
		t.Errorf("Point.Set(1) = %v err %v", q.Y, err)
	}
	for _, bad := range []int{-1, 2, 99} {
		if err := q.Set(bad, 0); err == nil {
			t.Errorf("Point.Set(%d) expected error", bad)
		}
	}
}

func TestPointUnitZero(t *testing.T) {
	if !NewPoint(0, 0).Unit().Equal(NewPoint(0, 0)) {
		t.Error("Unit of zero vector should be zero")
	}
	if !NewPoint(0, 0).AbsUnit().Equal(NewPoint(0, 0)) {
		t.Error("AbsUnit of zero vector should be zero")
	}
	// Non-zero unit vector has length 1 and AbsUnit has non-negative comps.
	u := NewPoint(3, 4).Unit()
	if !feq(math.Hypot(u.X, u.Y), 1) {
		t.Errorf("Unit length = %v, want 1", math.Hypot(u.X, u.Y))
	}
	a := NewPoint(-3, -4).AbsUnit()
	if a.X < 0 || a.Y < 0 {
		t.Errorf("AbsUnit should be non-negative, got %v", a.Tuple())
	}
}

func TestRectWidthHeight(t *testing.T) {
	r := NewRect(10, 20, 40, 80)
	if r.Width() != 30 {
		t.Errorf("Width = %v, want 30", r.Width())
	}
	if r.Height() != 60 {
		t.Errorf("Height = %v, want 60", r.Height())
	}
}

func TestRectNormalize(t *testing.T) {
	cases := []struct {
		in, want Rect
	}{
		{NewRect(0, 0, 10, 10), NewRect(0, 0, 10, 10)}, // already normal: no swaps
		{NewRect(10, 0, 0, 10), NewRect(0, 0, 10, 10)}, // swap x
		{NewRect(0, 10, 10, 0), NewRect(0, 0, 10, 10)}, // swap y
		{NewRect(10, 10, 0, 0), NewRect(0, 0, 10, 10)}, // swap both
	}
	for _, c := range cases {
		if got := c.in.Normalize(); !got.Equal(c.want) {
			t.Errorf("Normalize(%v) = %v, want %v", c.in.Tuple(), got.Tuple(), c.want.Tuple())
		}
	}
}

func TestRectContainsRect(t *testing.T) {
	outer := NewRect(0, 0, 100, 100)
	if !outer.ContainsRect(NewRect(10, 10, 90, 90)) {
		t.Error("inner rect should be contained")
	}
	if !outer.ContainsRect(outer) {
		t.Error("rect should contain itself")
	}
	if outer.ContainsRect(NewRect(-1, 10, 50, 50)) {
		t.Error("rect crossing left edge should not be contained")
	}
	if outer.ContainsRect(NewRect(10, 10, 101, 50)) {
		t.Error("rect crossing right edge should not be contained")
	}
}

func TestRectGet(t *testing.T) {
	r := NewRect(1, 2, 3, 4)
	want := []float64{1, 2, 3, 4}
	for i, w := range want {
		got, err := r.Get(i)
		if err != nil {
			t.Fatalf("Rect.Get(%d) err = %v", i, err)
		}
		if got != w {
			t.Errorf("Rect.Get(%d) = %v, want %v", i, got, w)
		}
	}
	for _, bad := range []int{-1, 4, 99} {
		if _, err := r.Get(bad); err == nil {
			t.Errorf("Rect.Get(%d) expected error", bad)
		}
	}
	if err := (&Rect{}).Set(-1, 0); err == nil {
		t.Error("Rect.Set(-1) expected error")
	}
}

func TestIRectRectAndGet(t *testing.T) {
	r := NewIRect(1, 2, 3, 4)
	if !r.Rect().Equal(NewRect(1, 2, 3, 4)) {
		t.Errorf("IRect.Rect = %v", r.Rect().Tuple())
	}
	want := []int{1, 2, 3, 4}
	for i, w := range want {
		got, err := r.Get(i)
		if err != nil {
			t.Fatalf("IRect.Get(%d) err = %v", i, err)
		}
		if got != w {
			t.Errorf("IRect.Get(%d) = %v, want %v", i, got, w)
		}
	}
	for _, bad := range []int{-1, 4, 99} {
		if _, err := r.Get(bad); err == nil {
			t.Errorf("IRect.Get(%d) expected error", bad)
		}
	}
	if err := (&IRect{}).Set(-1, 0); err == nil {
		t.Error("IRect.Set(-1) expected error")
	}
}

func TestIRectIncludeRectInvalid(t *testing.T) {
	r := NewIRect(10, 20, 100, 200)
	// X0 >= X1 empty branch.
	if !r.IncludeRect(NewIRect(5, 5, 5, 10)).Equal(r) {
		t.Error("include x-empty irect should be no-op")
	}
	// Y0 >= Y1 empty branch.
	if !r.IncludeRect(NewIRect(5, 5, 10, 5)).Equal(r) {
		t.Error("include y-empty irect should be no-op")
	}
}

func TestMatrixGetSet(t *testing.T) {
	m := NewMatrix(1, 2, 3, 4, 5, 6)
	want := []float64{1, 2, 3, 4, 5, 6}
	for i, w := range want {
		got, err := m.Get(i)
		if err != nil {
			t.Fatalf("Matrix.Get(%d) err = %v", i, err)
		}
		if got != w {
			t.Errorf("Matrix.Get(%d) = %v, want %v", i, got, w)
		}
	}
	for _, bad := range []int{-1, 6, 99} {
		if _, err := m.Get(bad); err == nil {
			t.Errorf("Matrix.Get(%d) expected error", bad)
		}
	}
	if err := (&Matrix{}).Set(-1, 0); err == nil {
		t.Error("Matrix.Set(-1) expected error")
	}
	if err := (&Matrix{}).Set(99, 0); err == nil {
		t.Error("Matrix.Set(99) expected error")
	}
}

func TestMatrixInvert(t *testing.T) {
	// Invertible: Invert returns true and round-trips to identity.
	m := Scale(2, 4)
	inv := m
	if !inv.Invert() {
		t.Fatal("Invert(scale) should succeed")
	}
	if inv.Mul(m).Sub(Identity).Abs() >= EPSILON {
		t.Errorf("inv*m != Identity: %v", inv.Mul(m).Tuple())
	}
	// Mathematical correctness: Inverted of Scale(2,2) halves a point.
	half := Scale(2, 2).Inverted()
	if !NewPoint(8, 6).Transform(half).Equal(NewPoint(4, 3)) {
		t.Error("inverse scale should halve point")
	}
	// Non-invertible: Invert returns false, matrix unchanged.
	bad := NewMatrix(1, 1, 1, 1, 0, 0) // det = 0
	orig := bad
	if bad.Invert() {
		t.Error("Invert(singular) should fail")
	}
	if bad != orig {
		t.Error("Invert failure should leave matrix unchanged")
	}
}

func TestScaleTransform(t *testing.T) {
	// Mathematical correctness: Scale(2,2) doubles a point.
	if !NewPoint(3, 5).Transform(Scale(2, 2)).Equal(NewPoint(6, 10)) {
		t.Error("Scale(2,2) should double point")
	}
}

func TestSinCosSnapQuadrants(t *testing.T) {
	// Hits negative wraparound (deg<0), >=360 wraparound, and all four snaps.
	cases := []struct {
		deg, c, s float64
	}{
		{0, 1, 0},
		{90, 0, 1},
		{180, -1, 0},
		{270, 0, -1},
		{360, 1, 0},  // wraps to 0
		{-90, 0, -1}, // wraps to 270
		{450, 0, 1},  // wraps to 90
		{-270, 0, 1}, // wraps to 90
	}
	for _, c := range cases {
		m := Rotate(c.deg)
		if !feq(m.A, c.c) || !feq(m.B, c.s) {
			t.Errorf("Rotate(%v) = c%v s%v, want c%v s%v", c.deg, m.A, m.B, c.c, c.s)
		}
	}
	// Non-quadrant angle exercises the trig fall-through path.
	m := Rotate(30)
	if !feq(m.A, math.Cos(math.Pi/6)) || !feq(m.B, math.Sin(math.Pi/6)) {
		t.Errorf("Rotate(30) = %v", m.Tuple())
	}
}

func TestQuadRect(t *testing.T) {
	q := NewRect(10, 20, 30, 40).Quad()
	if !q.Rect().Equal(NewRect(10, 20, 30, 40)) {
		t.Errorf("Quad.Rect = %v", q.Rect().Tuple())
	}
}

func TestQuadGetSet(t *testing.T) {
	r := NewRect(0, 0, 10, 10)
	q := r.Quad()
	want := []Point{q.UL, q.UR, q.LL, q.LR}
	for i, w := range want {
		got, err := q.Get(i)
		if err != nil {
			t.Fatalf("Quad.Get(%d) err = %v", i, err)
		}
		if !got.Equal(w) {
			t.Errorf("Quad.Get(%d) = %v, want %v", i, got, w)
		}
	}
	for _, bad := range []int{-1, 4, 99} {
		if _, err := q.Get(bad); err == nil {
			t.Errorf("Quad.Get(%d) expected error", bad)
		}
	}
	var z Quad
	for i := 0; i < 4; i++ {
		if err := z.Set(i, NewPoint(float64(i), float64(i))); err != nil {
			t.Fatalf("Quad.Set(%d) err = %v", i, err)
		}
	}
	if !z.UL.Equal(NewPoint(0, 0)) || !z.UR.Equal(NewPoint(1, 1)) ||
		!z.LL.Equal(NewPoint(2, 2)) || !z.LR.Equal(NewPoint(3, 3)) {
		t.Errorf("Quad.Set build = %+v", z)
	}
	for _, bad := range []int{-1, 4, 99} {
		if err := z.Set(bad, Point{}); err == nil {
			t.Errorf("Quad.Set(%d) expected error", bad)
		}
	}
}

func TestQuadConvexContains(t *testing.T) {
	// Square: convex, contains its center and corners, contains a sub-rect.
	q := NewRect(0, 0, 10, 10).Quad()
	if !q.IsConvex() {
		t.Error("square should be convex")
	}
	if !q.ContainsPoint(NewPoint(5, 5)) {
		t.Error("center should be contained")
	}
	if !q.ContainsRect(NewRect(2, 2, 8, 8)) {
		t.Error("sub-rect should be contained")
	}
	if !q.ContainsQuad(NewRect(2, 2, 8, 8).Quad()) {
		t.Error("sub-quad should be contained")
	}
	if q.ContainsPoint(NewPoint(20, 20)) {
		t.Error("far point should not be contained")
	}
	if q.ContainsRect(NewRect(2, 2, 20, 8)) {
		t.Error("rect poking out should not be contained")
	}
	if q.ContainsQuad(NewRect(-5, -5, 5, 5).Quad()) {
		t.Error("overlapping quad should not be contained")
	}

	// Non-convex (self-intersecting) quad: swap two corners so traversal
	// UL->UR->LR->LL produces a bow-tie, flipping the cross-product sign.
	bow := Quad{
		UL: NewPoint(0, 0),
		UR: NewPoint(10, 10),
		LL: NewPoint(0, 10),
		LR: NewPoint(10, 0),
	}
	if bow.IsConvex() {
		t.Error("bow-tie quad should not be convex")
	}

	// Degenerate quad with three collinear consecutive corners exercises the
	// near-zero cross-product skip in IsConvex.
	collinear := Quad{
		UL: NewPoint(0, 0),
		UR: NewPoint(5, 0),
		LR: NewPoint(10, 0),
		LL: NewPoint(10, 10),
	}
	if !collinear.IsConvex() {
		t.Error("quad with collinear edge should still count as convex")
	}

	// A point lying exactly on an edge exercises the on-edge skip in
	// ContainsPoint.
	if !q.ContainsPoint(NewPoint(5, 0)) {
		t.Error("point on edge should be contained")
	}
}

func TestIminImax(t *testing.T) {
	// Cover both branches of imin/imax via IRect.IncludePoint.
	r := NewIRect(5, 5, 5, 5)
	// Point below/left -> imin takes the point, imax keeps r.
	if got := r.IncludePoint(NewPoint(1, 1)).Tuple(); got != [4]int{1, 1, 5, 5} {
		t.Errorf("IncludePoint(low) = %v", got)
	}
	// Point above/right -> imin keeps r, imax takes the point.
	if got := r.IncludePoint(NewPoint(9, 9)).Tuple(); got != [4]int{5, 5, 9, 9} {
		t.Errorf("IncludePoint(high) = %v", got)
	}
}
