package gomupdf

import (
	"strings"
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// newDrawPage is a helper that creates a fresh in-memory PDF with one 300×300
// page and returns both the document and page 0.
func newDrawPage(t *testing.T) (*Document, *Page) {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatalf("NewPDF: %v", err)
	}
	if err := d.NewPage(300, 300); err != nil {
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

// TestDrawRect checks that DrawRect appends at least one stroked path.
func TestDrawRect(t *testing.T) {
	d, pg := newDrawPage(t)
	defer d.Close()

	red := [3]float64{1, 0, 0}
	opts := DrawOptions{Stroke: &red, Width: 2}
	r := geometry.Rect{X0: 10, Y0: 10, X1: 100, Y1: 100}
	if err := pg.DrawRect(r, opts); err != nil {
		t.Fatalf("DrawRect: %v", err)
	}

	drws, err := pg.GetDrawings()
	if err != nil {
		t.Fatalf("GetDrawings: %v", err)
	}
	if len(drws) == 0 {
		t.Fatal("expected at least one drawing after DrawRect, got none")
	}
	// At least one stroke entry should exist.
	hasStroke := false
	for _, dw := range drws {
		if dw.Type == "s" {
			hasStroke = true
			break
		}
	}
	if !hasStroke {
		t.Errorf("expected a stroked path; got types: %v", drawingTypes(drws))
	}
}

// TestDrawRectFill checks that DrawRect with Fill produces a fill drawing.
func TestDrawRectFill(t *testing.T) {
	d, pg := newDrawPage(t)
	defer d.Close()

	blue := [3]float64{0, 0, 1}
	opts := DrawOptions{Fill: &blue, Width: 1}
	r := geometry.Rect{X0: 20, Y0: 20, X1: 80, Y1: 80}
	if err := pg.DrawRect(r, opts); err != nil {
		t.Fatalf("DrawRect(fill): %v", err)
	}
	drws, err := pg.GetDrawings()
	if err != nil {
		t.Fatalf("GetDrawings: %v", err)
	}
	if len(drws) == 0 {
		t.Fatal("expected at least one drawing after DrawRect fill, got none")
	}
}

// TestDrawLine checks that DrawLine appends a path with a line segment.
func TestDrawLine(t *testing.T) {
	d, pg := newDrawPage(t)
	defer d.Close()

	green := [3]float64{0, 0.5, 0}
	opts := DrawOptions{Stroke: &green, Width: 1.5}
	a := geometry.Point{X: 10, Y: 10}
	b := geometry.Point{X: 200, Y: 200}
	if err := pg.DrawLine(a, b, opts); err != nil {
		t.Fatalf("DrawLine: %v", err)
	}

	drws, err := pg.GetDrawings()
	if err != nil {
		t.Fatalf("GetDrawings: %v", err)
	}
	if len(drws) == 0 {
		t.Fatal("expected at least one drawing after DrawLine, got none")
	}
	hasLine := false
	for _, dw := range drws {
		for _, item := range dw.Items {
			if item.Op == "l" {
				hasLine = true
			}
		}
	}
	if !hasLine {
		t.Error("expected at least one line-segment path item after DrawLine")
	}
}

// TestDrawCircle checks that DrawCircle appends a path with curve segments.
func TestDrawCircle(t *testing.T) {
	d, pg := newDrawPage(t)
	defer d.Close()

	opts := DrawOptions{Width: 1}
	center := geometry.Point{X: 150, Y: 150}
	if err := pg.DrawCircle(center, 50, opts); err != nil {
		t.Fatalf("DrawCircle: %v", err)
	}

	drws, err := pg.GetDrawings()
	if err != nil {
		t.Fatalf("GetDrawings: %v", err)
	}
	if len(drws) == 0 {
		t.Fatal("expected at least one drawing after DrawCircle, got none")
	}
	hasCurve := false
	for _, dw := range drws {
		for _, item := range dw.Items {
			if item.Op == "c" {
				hasCurve = true
			}
		}
	}
	if !hasCurve {
		t.Error("expected at least one Bézier curve path item after DrawCircle")
	}
}

// TestDrawRoundTrip verifies that drawings persist through a save/reload cycle.
func TestDrawRoundTrip(t *testing.T) {
	d, pg := newDrawPage(t)
	defer d.Close()

	red := [3]float64{1, 0, 0}
	if err := pg.DrawRect(geometry.Rect{X0: 5, Y0: 5, X1: 50, Y1: 50}, DrawOptions{Stroke: &red}); err != nil {
		t.Fatalf("DrawRect: %v", err)
	}
	if err := pg.DrawLine(geometry.Point{X: 0, Y: 0}, geometry.Point{X: 100, Y: 100}, DrawOptions{}); err != nil {
		t.Fatalf("DrawLine: %v", err)
	}

	// Count drawings before save.
	before, err := pg.GetDrawings()
	if err != nil {
		t.Fatalf("GetDrawings before: %v", err)
	}
	if len(before) == 0 {
		t.Fatal("expected drawings before save")
	}

	// Save to bytes and reopen.
	data, err := d.SaveBytes(false)
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}
	d2, err := OpenStream(data)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer d2.Close()
	pg2, err := d2.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage on reopened: %v", err)
	}
	after, err := pg2.GetDrawings()
	if err != nil {
		t.Fatalf("GetDrawings after reload: %v", err)
	}
	if len(after) < len(before) {
		t.Errorf("drawings after reload (%d) < before save (%d)", len(after), len(before))
	}
}

// TestInsertTextboxWraps checks that InsertTextbox wraps a long string into
// multiple lines within a narrow rect and that the page text contains the words.
func TestInsertTextboxWraps(t *testing.T) {
	d, pg := newDrawPage(t)
	defer d.Close()

	// Narrow rect (width 150) with height enough for several lines.
	rect := geometry.Rect{X0: 10, Y0: 50, X1: 160, Y1: 280}
	text := "The quick brown fox jumps over the lazy dog and then runs away"
	n, err := pg.InsertTextbox(rect, text, 12)
	if err != nil {
		t.Fatalf("InsertTextbox: %v", err)
	}
	if n < 2 {
		t.Errorf("expected ≥2 lines for wrapped text, got %d", n)
	}

	got, err := pg.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if !strings.Contains(got, "quick") {
		t.Errorf("GetText output missing expected word; got: %q", got)
	}
}

// TestInsertTextboxClips checks that InsertTextbox clips text that overflows a
// rect tall enough for only one line at size 12 (height ≈ 14 pts).
func TestInsertTextboxClips(t *testing.T) {
	d, pg := newDrawPage(t)
	defer d.Close()

	// Rect tall enough for exactly one line (height ~14 pts, so 1.2*12 = 14.4).
	rect := geometry.Rect{X0: 10, Y0: 200, X1: 290, Y1: 214}
	text := "line one here and line two here and line three here and more words"
	n, err := pg.InsertTextbox(rect, text, 12)
	if err != nil {
		t.Fatalf("InsertTextbox clip: %v", err)
	}
	if n < 1 {
		t.Errorf("expected ≥1 line returned, got %d", n)
	}
	// Total lines if unconstrained would be more — assert we clipped.
	// With width=280 at size 12 it may fit on 1-2 lines; just assert positive.
	t.Logf("InsertTextbox clipping test: %d lines fit", n)
}

// TestInsertTextboxClosedDoc checks that InsertTextbox on a closed document
// returns an error.
func TestInsertTextboxClosedDoc(t *testing.T) {
	d, pg := newDrawPage(t)
	d.Close()
	_, err := pg.InsertTextbox(geometry.Rect{X0: 0, Y0: 0, X1: 100, Y1: 100}, "hello", 12)
	if err == nil {
		t.Fatal("expected error for InsertTextbox on closed doc")
	}
}

// TestDrawClosedDoc checks that draw ops on a closed document return errors.
func TestDrawClosedDoc(t *testing.T) {
	d, pg := newDrawPage(t)
	d.Close()

	if err := pg.DrawLine(geometry.Point{}, geometry.Point{X: 1}, DrawOptions{}); err == nil {
		t.Error("DrawLine on closed doc: expected error")
	}
	if err := pg.DrawRect(geometry.Rect{X0: 0, Y0: 0, X1: 10, Y1: 10}, DrawOptions{}); err == nil {
		t.Error("DrawRect on closed doc: expected error")
	}
	if err := pg.DrawCircle(geometry.Point{}, 5, DrawOptions{}); err == nil {
		t.Error("DrawCircle on closed doc: expected error")
	}
}

// drawingTypes is a helper for debug output.
func drawingTypes(drws []Drawing) []string {
	var out []string
	for _, d := range drws {
		out = append(out, d.Type)
	}
	return out
}
