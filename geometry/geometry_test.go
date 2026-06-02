package geometry

import "testing"

// Tests for the geometry package. Operator-style operations are exercised
// through their explicit Go methods. Cases that depend on page/pixmap features
// are deferred.

func TestRect(t *testing.T) {
	if got := NewRect(0, 0, 0, 0).Tuple(); got != [4]float64{0, 0, 0, 0} {
		t.Fatalf("empty rect tuple = %v", got)
	}
	p1 := NewPoint(10, 20)
	p2 := NewPoint(100, 200)
	p3 := NewPoint(150, 250)
	r := NewRect(10, 20, 100, 200)
	want := r.Tuple()
	if RectFromPoints(p1, p2).Tuple() != want {
		t.Error("RectFromPoints != Rect")
	}
	if got := r.IncludePoint(p3).Tuple(); got != [4]float64{10, 20, 150, 250} {
		t.Errorf("IncludePoint = %v", got)
	}
	r = NewRect(10, 20, 100, 200)
	if got := r.IncludeRect(NewRect(100, 200, 110, 220)).Tuple(); got != [4]float64{10, 20, 110, 220} {
		t.Errorf("IncludeRect = %v", got)
	}
	r = NewRect(10, 20, 100, 200)
	if r.IncludeRect(NewRect(0, 0, 0, 0)).Tuple() != want {
		t.Error("include empty rect should be no-op")
	}
	if r.IncludeRect(NewRect(1, 1, -1, -1)).Tuple() != want {
		t.Error("include invalid rect should be no-op")
	}
	r2 := NewRect(0, 0, 0, 0)
	for i := 0; i < 4; i++ {
		if err := r2.Set(i, float64(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	if !r2.Equal(NewRect(1, 2, 3, 4)) {
		t.Errorf("Set build = %v", r2.Tuple())
	}
	if !NewRect(0, 0, 0, 0).DivScalar(5).Equal(NewRect(0, 0, 0, 0)) {
		t.Error("empty/5 != empty")
	}
	if !NewRect(1, 1, 2, 2).DivMatrix(Identity).Equal(NewRect(1, 1, 2, 2)) {
		t.Error("rect/Identity != rect")
	}
	if err := (&Rect{}).Set(5, 1); err == nil {
		t.Error("Set(5) should error")
	}
}

func TestIRect(t *testing.T) {
	p1 := NewPoint(10, 20)
	p2 := NewPoint(100, 200)
	p3 := NewPoint(150, 250)
	r := NewIRect(10, 20, 100, 200)
	want := r.Tuple()
	if IRectFromPoints(p1, p2).Tuple() != want {
		t.Error("IRectFromPoints != IRect")
	}
	if got := r.IncludePoint(p3).Tuple(); got != [4]int{10, 20, 150, 250} {
		t.Errorf("IncludePoint = %v", got)
	}
	r = NewIRect(10, 20, 100, 200)
	if got := r.IncludeRect(NewIRect(100, 200, 110, 220)).Tuple(); got != [4]int{10, 20, 110, 220} {
		t.Errorf("IncludeRect = %v", got)
	}
	r = NewIRect(10, 20, 100, 200)
	if r.IncludeRect(NewIRect(0, 0, 0, 0)).Tuple() != want {
		t.Error("include empty irect should be no-op")
	}
	r2 := NewIRect(0, 0, 0, 0)
	for i := 0; i < 4; i++ {
		if err := r2.Set(i, i+1); err != nil {
			t.Fatal(err)
		}
	}
	if !r2.Equal(NewIRect(1, 2, 3, 4)) {
		t.Errorf("Set build = %v", r2.Tuple())
	}
	if err := (&IRect{}).Set(5, 1); err == nil {
		t.Error("Set(5) should error")
	}
}

func TestInversion(t *testing.T) {
	m1 := Rotate(255)
	m2 := Rotate(-255)
	m3 := m1.Mul(m2)
	if m3.Sub(Identity).Abs() >= EPSILON {
		t.Errorf("m1*m2 != Identity: %v", m3.Tuple())
	}
	m := NewMatrix(1, 0, 1, 0, 1, 0) // not invertible
	if !m.Inverted().Equal(NewMatrix(0, 0, 0, 0, 0, 0)) {
		t.Errorf("non-invertible inverse should be zero, got %v", m.Inverted().Tuple())
	}
}

func TestMatrix(t *testing.T) {
	if NewMatrix(0, 0, 0, 0, 0, 0).Tuple() != [6]float64{0, 0, 0, 0, 0, 0} {
		t.Error("empty matrix")
	}
	if Rotate(90).Tuple() != [6]float64{0, 1, -1, 0, 0, 0} {
		t.Errorf("Rotate(90) = %v", Rotate(90).Tuple())
	}
	m45p := Rotate(45)
	m45m := Rotate(-45)
	m90 := Rotate(90)
	if m90.Sub(m45p.Mul(m45p)).Abs() >= EPSILON {
		t.Error("m90 != m45*m45")
	}
	if Identity.Sub(m45p.Mul(m45m)).Abs() >= EPSILON {
		t.Error("Identity != m45p*m45m")
	}
	if m45p.Sub(m45m.Inverted()).Abs() >= EPSILON {
		t.Error("m45p != ~m45m")
	}
	if !Shear(2, 3).Equal(NewMatrix(1, 3, 2, 1, 0, 0)) {
		t.Errorf("Shear(2,3) = %v", Shear(2, 3).Tuple())
	}
	m := Shear(2, 3)
	m.Invert()
	if m.Mul(Shear(2, 3)).Sub(Identity).Abs() >= EPSILON {
		t.Error("invert(shear)*shear != Identity")
	}
	if !Scale(1, 1).Pretranslate(2, 3).Equal(NewMatrix(1, 0, 0, 1, 2, 3)) {
		t.Error("pretranslate")
	}
	if !Scale(1, 1).Prescale(2, 3).Equal(NewMatrix(2, 0, 0, 3, 0, 0)) {
		t.Error("prescale")
	}
	if !Scale(1, 1).Preshear(2, 3).Equal(NewMatrix(1, 3, 2, 1, 0, 0)) {
		t.Error("preshear")
	}
	if Scale(1, 1).Prerotate(30).Sub(Rotate(30)).Abs() >= EPSILON {
		t.Error("prerotate(30)")
	}
	const small = 1e-6
	if !Scale(1, 1).Prerotate(90 + small).Equal(Rotate(90)) {
		t.Error("prerotate(90+small) snap")
	}
	if !Scale(1, 1).Prerotate(180 + small).Equal(Rotate(180)) {
		t.Error("prerotate(180+small) snap")
	}
	if !Scale(1, 1).Prerotate(270 + small).Equal(Rotate(270)) {
		t.Error("prerotate(270+small) snap")
	}
	if !Scale(1, 1).Prerotate(small).Equal(Rotate(0)) {
		t.Error("prerotate(small) snap")
	}
	if !Scale(1, 1).Concat(Scale(1, 2), Scale(3, 4)).Equal(NewMatrix(3, 0, 0, 8, 0, 0)) {
		t.Error("concat")
	}
	if !NewMatrix(1, 2, 3, 4, 5, 6).DivScalar(1).Equal(NewMatrix(1, 2, 3, 4, 5, 6)) {
		t.Error("matrix/1")
	}
	mm := NewMatrix(1, 2, 3, 4, 5, 6)
	g0, _ := mm.Get(0)
	g5, _ := mm.Get(5)
	if g0 != mm.A || g5 != mm.F {
		t.Error("Get index")
	}
	m2 := NewMatrix(0, 0, 0, 0, 0, 0)
	for i := 0; i < 6; i++ {
		if err := m2.Set(i, float64(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	if !m2.Equal(NewMatrix(1, 2, 3, 4, 5, 6)) {
		t.Error("Set build matrix")
	}
}

func TestPoint(t *testing.T) {
	if NewPoint(0, 0).Tuple() != [2]float64{0, 0} {
		t.Error("empty point")
	}
	if !NewPoint(1, -1).Unit().Equal(NewPoint(5, -5).Unit()) {
		t.Error("unit")
	}
	if !NewPoint(-1, -1).AbsUnit().Equal(NewPoint(1, 1).Unit()) {
		t.Error("abs_unit")
	}
	if NewPoint(1, 1).DistanceToPoint(NewPoint(1, 1)) != 0 {
		t.Error("distance to self")
	}
	if NewPoint(1, 1).DistanceToRect(NewRect(1, 1, 2, 2)) != 0 {
		t.Error("distance to containing rect")
	}
	if NewPoint(0, 0).DistanceToRect(NewRect(1, 1, 2, 2)) <= 0 {
		t.Error("distance to far rect")
	}
	if err := (&Point{}).Set(3, 1); err == nil {
		t.Error("Set(3) should error")
	}
}

func TestAlgebra(t *testing.T) {
	p := NewPoint(1, 2)
	m := NewMatrix(1, 2, 3, 4, 5, 6)
	r := NewRect(1, 1, 2, 2)
	if !p.Add(p).Equal(p.Mul(2)) {
		t.Error("p+p")
	}
	if !p.Sub(p).Equal(NewPoint(0, 0)) {
		t.Error("p-p")
	}
	if !m.Add(m).Equal(m.MulScalar(2)) {
		t.Error("m+m")
	}
	if !m.Sub(m).Equal(NewMatrix(0, 0, 0, 0, 0, 0)) {
		t.Error("m-m")
	}
	if !r.Add(r).Equal(r.Mul(2)) {
		t.Error("r+r")
	}
	if !r.Sub(r).Equal(NewRect(0, 0, 0, 0)) {
		t.Error("r-r")
	}
	if !p.AddScalar(5).Equal(NewPoint(6, 7)) {
		t.Error("p+5")
	}
	if !m.AddScalar(5).Equal(NewMatrix(6, 7, 8, 9, 10, 11)) {
		t.Error("m+5")
	}
	if !r.ContainsPoint(r.TL()) {
		t.Error("tl in r")
	}
	if r.ContainsPoint(r.TR()) || r.ContainsPoint(r.BR()) || r.ContainsPoint(r.BL()) {
		t.Error("tr/br/bl should not be in r (half-open)")
	}
	if !p.Transform(m).Equal(NewPoint(12, 16)) {
		t.Errorf("p*m = %v", p.Transform(m))
	}
	if !r.Transform(m).Equal(NewRect(9, 12, 13, 18)) {
		t.Errorf("r*m = %v", r.Transform(m).Tuple())
	}
	if !NewRect(1, 1, 2, 2).Intersect(NewRect(2, 2, 3, 3)).IsEmpty() {
		t.Error("disjoint intersect should be empty")
	}
	if NewRect(1, 1, 2, 2).Intersects(NewRect(2, 2, 4, 4)) {
		t.Error("edge-touch should not intersect")
	}
}

func TestQuad(t *testing.T) {
	r := NewRect(10, 10, 20, 20)
	q := r.Quad()
	if !q.IsRectangular() {
		t.Error("rect quad should be rectangular")
	}
	if q.IsEmpty() {
		t.Error("should not be empty")
	}
	if !q.IsConvex() {
		t.Error("should be convex")
	}
	q = q.Transform(Scale(1, 1).Preshear(2, 3))
	if q.IsRectangular() {
		t.Error("sheared quad should not be rectangular")
	}
	if q.IsEmpty() {
		t.Error("sheared quad should not be empty")
	}
	if !q.IsConvex() {
		t.Error("sheared quad should be convex")
	}
	if q.ContainsPoint(r.TL()) {
		t.Error("r.tl should not be in sheared q")
	}
	if q.ContainsRect(r) {
		t.Error("r should not be in sheared q")
	}
	if q.ContainsQuad(r.Quad()) {
		t.Error("r.quad should not be in sheared q")
	}
}
