package gomupdf

import (
	"bytes"
	"image/png"
	"testing"
)

// Ported concepts from test_pixmap.py — render a page and validate the raster's
// dimensions, component count, stride invariant, zoom scaling, and PNG output.
func TestPixmapRGB(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	pm, err := p.Pixmap()
	if err != nil {
		t.Fatal(err)
	}
	if pm.Width <= 0 || pm.Height <= 0 {
		t.Fatalf("bad dims %dx%d", pm.Width, pm.Height)
	}
	if pm.N != 3 {
		t.Errorf("rgb pixmap N = %d, want 3", pm.N)
	}
	if pm.Stride != pm.Width*pm.N {
		t.Errorf("stride %d != width*n %d", pm.Stride, pm.Width*pm.N)
	}
	if len(pm.Samples) != pm.Stride*pm.Height {
		t.Errorf("samples %d != stride*height %d", len(pm.Samples), pm.Stride*pm.Height)
	}

	// PNG round-trips to the same dimensions.
	data, err := pm.PNG()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != pm.Width || b.Dy() != pm.Height {
		t.Errorf("png %dx%d != pixmap %dx%d", b.Dx(), b.Dy(), pm.Width, pm.Height)
	}
}

func TestPixmapZoomScales(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	one, _ := p.Pixmap(PixmapOptions{Zoom: 1})
	two, _ := p.Pixmap(PixmapOptions{Zoom: 2})
	if two.Width != one.Width*2 || two.Height != one.Height*2 {
		t.Errorf("zoom 2 = %dx%d, want %dx%d", two.Width, two.Height, one.Width*2, one.Height*2)
	}
}

func TestPixmapGrayAndAlpha(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	gray, err := p.Pixmap(PixmapOptions{Gray: true})
	if err != nil {
		t.Fatal(err)
	}
	if gray.N != 1 {
		t.Errorf("gray N = %d, want 1", gray.N)
	}

	rgba, err := p.Pixmap(PixmapOptions{Alpha: true})
	if err != nil {
		t.Fatal(err)
	}
	if rgba.N != 4 || !rgba.Alpha {
		t.Errorf("rgba N = %d alpha=%v, want 4/true", rgba.N, rgba.Alpha)
	}
	// a pixel must return N bytes
	if px := rgba.Pixel(0, 0); len(px) != 4 {
		t.Errorf("Pixel returned %d bytes, want 4", len(px))
	}
}
