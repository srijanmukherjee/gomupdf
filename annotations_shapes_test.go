package gomupdf

import (
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// newShapeTestPage creates a fresh 400x400 PDF and returns the doc and page 0.
// Caller is responsible for closing the document.
func newShapeTestPage(t *testing.T) (*Document, *Page) {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatalf("NewPDF: %v", err)
	}
	if err := d.NewPage(400, 400); err != nil {
		d.Close()
		t.Fatalf("NewPage: %v", err)
	}
	pg, err := d.LoadPage(0)
	if err != nil {
		d.Close()
		t.Fatalf("LoadPage: %v", err)
	}
	return d, pg
}

// annotsByType returns all annotations of the given type string.
func annotsByType(t *testing.T, pg *Page, want string) []Annotation {
	t.Helper()
	annots, err := pg.Annotations()
	if err != nil {
		t.Fatalf("Annotations: %v", err)
	}
	var out []Annotation
	for _, a := range annots {
		if a.Type == want {
			out = append(out, a)
		}
	}
	return out
}

// reopenPage serializes doc to bytes and reopens the first page.
func reopenPage(t *testing.T, d *Document) (*Document, *Page) {
	t.Helper()
	b, err := d.SaveBytes(false)
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}
	d2, err := OpenStream(b)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	pg2, err := d2.LoadPage(0)
	if err != nil {
		d2.Close()
		t.Fatalf("LoadPage (reopen): %v", err)
	}
	return d2, pg2
}

func TestAddLine(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	a := geometry.Point{X: 10, Y: 10}
	b := geometry.Point{X: 200, Y: 200}
	if err := pg.AddLine(a, b, AnnotStyle{}); err != nil {
		t.Fatalf("AddLine: %v", err)
	}

	got := annotsByType(t, pg, "Line")
	if len(got) != 1 {
		// Log all to help adapt if type string differs
		all, _ := pg.Annotations()
		for _, x := range all {
			t.Logf("annotation type=%q", x.Type)
		}
		t.Fatalf("expected 1 Line annotation, got %d", len(got))
	}
}

func TestAddCircle(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	red := [3]float64{1, 0, 0}
	blue := [3]float64{0, 0, 1}
	style := AnnotStyle{
		Stroke:  &red,
		Fill:    &blue,
		Width:   2.5,
		Opacity: 0.8,
	}
	r := geometry.Rect{X0: 50, Y0: 50, X1: 150, Y1: 150}
	if err := pg.AddCircle(r, style); err != nil {
		t.Fatalf("AddCircle: %v", err)
	}

	got := annotsByType(t, pg, "Circle")
	if len(got) != 1 {
		all, _ := pg.Annotations()
		for _, x := range all {
			t.Logf("annotation type=%q", x.Type)
		}
		t.Fatalf("expected 1 Circle annotation, got %d", len(got))
	}
}

func TestAddCircleRoundTrip(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	red := [3]float64{1, 0, 0}
	blue := [3]float64{0, 0, 1}
	style := AnnotStyle{Stroke: &red, Fill: &blue, Width: 2, Opacity: 0.9}
	r := geometry.Rect{X0: 20, Y0: 20, X1: 80, Y1: 80}
	if err := pg.AddCircle(r, style); err != nil {
		t.Fatalf("AddCircle: %v", err)
	}

	d2, pg2 := reopenPage(t, d)
	defer d2.Close()

	got := annotsByType(t, pg2, "Circle")
	if len(got) != 1 {
		all, _ := pg2.Annotations()
		for _, x := range all {
			t.Logf("annotation type=%q", x.Type)
		}
		t.Fatalf("after reopen: expected 1 Circle, got %d", len(got))
	}
}

func TestAddPolygon(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	pts := []geometry.Point{
		{X: 50, Y: 50},
		{X: 150, Y: 50},
		{X: 100, Y: 150},
	}
	if err := pg.AddPolygon(pts, AnnotStyle{}); err != nil {
		t.Fatalf("AddPolygon: %v", err)
	}

	got := annotsByType(t, pg, "Polygon")
	if len(got) != 1 {
		all, _ := pg.Annotations()
		for _, x := range all {
			t.Logf("annotation type=%q", x.Type)
		}
		t.Fatalf("expected 1 Polygon annotation, got %d", len(got))
	}
}

func TestAddPolygonTooFewPoints(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	err := pg.AddPolygon([]geometry.Point{{X: 1, Y: 1}}, AnnotStyle{})
	if err == nil {
		t.Fatal("expected error for single-point polygon, got nil")
	}
}

func TestAddPolyline(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	pts := []geometry.Point{
		{X: 10, Y: 100},
		{X: 100, Y: 50},
		{X: 200, Y: 100},
		{X: 300, Y: 50},
	}
	if err := pg.AddPolyline(pts, AnnotStyle{}); err != nil {
		t.Fatalf("AddPolyline: %v", err)
	}

	got := annotsByType(t, pg, "PolyLine")
	if len(got) != 1 {
		all, _ := pg.Annotations()
		for _, x := range all {
			t.Logf("annotation type=%q", x.Type)
		}
		t.Fatalf("expected 1 PolyLine annotation, got %d", len(got))
	}
}

func TestAddInk(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	strokes := [][]geometry.Point{
		{{X: 10, Y: 10}, {X: 50, Y: 80}, {X: 100, Y: 30}},
		{{X: 150, Y: 150}, {X: 200, Y: 200}},
	}
	green := [3]float64{0, 0.8, 0}
	style := AnnotStyle{Stroke: &green, Width: 3, Opacity: 1}
	if err := pg.AddInk(strokes, style); err != nil {
		t.Fatalf("AddInk: %v", err)
	}

	got := annotsByType(t, pg, "Ink")
	if len(got) != 1 {
		all, _ := pg.Annotations()
		for _, x := range all {
			t.Logf("annotation type=%q", x.Type)
		}
		t.Fatalf("expected 1 Ink annotation, got %d", len(got))
	}
}

func TestAddInkEmpty(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	err := pg.AddInk(nil, AnnotStyle{})
	if err == nil {
		t.Fatal("expected error for empty ink strokes, got nil")
	}
}

func TestAddFreeText(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	r := geometry.Rect{X0: 50, Y0: 50, X1: 300, Y1: 120}
	const msg = "Hello from FreeText"
	if err := pg.AddFreeText(r, msg, 14, AnnotStyle{}); err != nil {
		t.Fatalf("AddFreeText: %v", err)
	}

	got := annotsByType(t, pg, "FreeText")
	if len(got) != 1 {
		all, _ := pg.Annotations()
		for _, x := range all {
			t.Logf("annotation type=%q contents=%q", x.Type, x.Contents)
		}
		t.Fatalf("expected 1 FreeText annotation, got %d", len(got))
	}
	if got[0].Contents != msg {
		t.Errorf("FreeText contents: got %q, want %q", got[0].Contents, msg)
	}
}

func TestAddFreeTextContentsRoundTrip(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	r := geometry.Rect{X0: 10, Y0: 10, X1: 300, Y1: 80}
	const msg = "Persisted content"
	if err := pg.AddFreeText(r, msg, 12, AnnotStyle{}); err != nil {
		t.Fatalf("AddFreeText: %v", err)
	}

	d2, pg2 := reopenPage(t, d)
	defer d2.Close()

	got := annotsByType(t, pg2, "FreeText")
	if len(got) != 1 {
		t.Fatalf("after reopen: expected 1 FreeText, got %d", len(got))
	}
	if got[0].Contents != msg {
		t.Errorf("after reopen FreeText contents: got %q, want %q", got[0].Contents, msg)
	}
}

func TestAddTextNote(t *testing.T) {
	d, pg := newShapeTestPage(t)
	defer d.Close()

	const note = "sticky note content"
	if err := pg.AddTextNote(geometry.Point{X: 100, Y: 100}, note); err != nil {
		t.Fatalf("AddTextNote: %v", err)
	}

	got := annotsByType(t, pg, "Text")
	if len(got) != 1 {
		all, _ := pg.Annotations()
		for _, x := range all {
			t.Logf("annotation type=%q", x.Type)
		}
		t.Fatalf("expected 1 Text annotation, got %d", len(got))
	}
}

func TestAddShapesClosedDoc(t *testing.T) {
	d, pg := newShapeTestPage(t)
	d.Close()

	if err := pg.AddLine(geometry.Point{}, geometry.Point{X: 1, Y: 1}, AnnotStyle{}); err == nil {
		t.Error("AddLine on closed doc: expected error, got nil")
	}
	if err := pg.AddCircle(geometry.Rect{X0: 0, Y0: 0, X1: 10, Y1: 10}, AnnotStyle{}); err == nil {
		t.Error("AddCircle on closed doc: expected error, got nil")
	}
	if err := pg.AddPolygon([]geometry.Point{{}, {X: 1}}, AnnotStyle{}); err == nil {
		t.Error("AddPolygon on closed doc: expected error, got nil")
	}
	if err := pg.AddPolyline([]geometry.Point{{}, {X: 1}}, AnnotStyle{}); err == nil {
		t.Error("AddPolyline on closed doc: expected error, got nil")
	}
	if err := pg.AddInk([][]geometry.Point{{{}, {X: 1}}}, AnnotStyle{}); err == nil {
		t.Error("AddInk on closed doc: expected error, got nil")
	}
	if err := pg.AddFreeText(geometry.Rect{X0: 0, Y0: 0, X1: 100, Y1: 50}, "x", 12, AnnotStyle{}); err == nil {
		t.Error("AddFreeText on closed doc: expected error, got nil")
	}
	if err := pg.AddTextNote(geometry.Point{}, "x"); err == nil {
		t.Error("AddTextNote on closed doc: expected error, got nil")
	}
}
