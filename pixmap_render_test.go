package gomupdf

import (
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// TestPixmapDPI verifies that rendering at 144 DPI produces roughly double the
// pixel dimensions of 72 DPI (within ±2px per axis).
func TestPixmapDPI(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	pm72, err := p.Pixmap(PixmapOptions{DPI: 72})
	if err != nil {
		t.Fatalf("DPI=72: %v", err)
	}
	pm144, err := p.Pixmap(PixmapOptions{DPI: 144})
	if err != nil {
		t.Fatalf("DPI=144: %v", err)
	}

	wantW := pm72.Width * 2
	wantH := pm72.Height * 2
	if iabs(pm144.Width-wantW) > 2 || iabs(pm144.Height-wantH) > 2 {
		t.Errorf("DPI=144 dims %dx%d, want ~%dx%d (±2)", pm144.Width, pm144.Height, wantW, wantH)
	}
}

// TestPixmapClip verifies that clipping to the top half of the page yields a
// pixmap whose height is roughly half that of the full render.
func TestPixmapClip(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	full, err := p.Pixmap(PixmapOptions{Zoom: 1})
	if err != nil {
		t.Fatalf("full render: %v", err)
	}

	// page bound in PDF points
	bound, err := p.Bound()
	if err != nil {
		t.Fatalf("bound: %v", err)
	}
	midY := (bound.Y0 + bound.Y1) / 2
	clip := &geometry.Rect{X0: bound.X0, Y0: bound.Y0, X1: bound.X1, Y1: midY}

	clipped, err := p.Pixmap(PixmapOptions{Zoom: 1, Clip: clip})
	if err != nil {
		t.Fatalf("clipped render: %v", err)
	}

	// clipped height should be ~half full height (allow ±2px)
	wantH := full.Height / 2
	if iabs(clipped.Height-wantH) > 2 {
		t.Errorf("clip height = %d, want ~%d (±2)", clipped.Height, wantH)
	}
	if clipped.Width != full.Width {
		t.Errorf("clip width = %d, want %d (same as full)", clipped.Width, full.Width)
	}
}

// TestPixmapCMYK verifies CMYK rendering returns N==4 (without alpha) or N==5
// (with alpha, since CMYK has 4 ink channels + 1 alpha channel).
func TestPixmapCMYK(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	pm, err := p.Pixmap(PixmapOptions{CMYK: true})
	if err != nil {
		t.Fatalf("CMYK render: %v", err)
	}
	if pm.N != 4 {
		t.Errorf("CMYK N = %d, want 4", pm.N)
	}
	if pm.Width <= 0 || pm.Height <= 0 {
		t.Errorf("bad dims %dx%d", pm.Width, pm.Height)
	}
	if len(pm.Samples) == 0 {
		t.Error("empty samples")
	}
}

// TestPixmapColorspaceN verifies component counts across colorspaces.
func TestPixmapColorspaceN(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	rgb, err := p.Pixmap(PixmapOptions{})
	if err != nil {
		t.Fatalf("rgb: %v", err)
	}
	if rgb.N != 3 {
		t.Errorf("RGB N = %d, want 3", rgb.N)
	}

	gray, err := p.Pixmap(PixmapOptions{Gray: true})
	if err != nil {
		t.Fatalf("gray: %v", err)
	}
	if gray.N != 1 {
		t.Errorf("Gray N = %d, want 1", gray.N)
	}

	rgbA, err := p.Pixmap(PixmapOptions{Alpha: true})
	if err != nil {
		t.Fatalf("rgb+alpha: %v", err)
	}
	if rgbA.N != 4 {
		t.Errorf("RGB+Alpha N = %d, want 4", rgbA.N)
	}

	grayA, err := p.Pixmap(PixmapOptions{Gray: true, Alpha: true})
	if err != nil {
		t.Fatalf("gray+alpha: %v", err)
	}
	if grayA.N != 2 {
		t.Errorf("Gray+Alpha N = %d, want 2", grayA.N)
	}
}

// TestPixmapNoAnnots verifies that rendering with NoAnnots produces a valid,
// non-empty pixmap without error.
func TestPixmapNoAnnots(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	pm, err := p.Pixmap(PixmapOptions{NoAnnots: true})
	if err != nil {
		t.Fatalf("NoAnnots render: %v", err)
	}
	if pm.Width <= 0 || pm.Height <= 0 {
		t.Errorf("bad dims %dx%d", pm.Width, pm.Height)
	}
	if len(pm.Samples) == 0 {
		t.Error("empty samples")
	}
}

// TestPixmapRenderClosedDoc verifies that rendering on a closed document
// returns an error.
func TestPixmapRenderClosedDoc(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	p, _ := d.LoadPage(0)
	d.Close()

	_, err := p.Pixmap(PixmapOptions{DPI: 72})
	if err == nil {
		t.Fatal("expected error rendering page of closed document")
	}
}

// iabs returns the absolute value of an int.
func iabs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
