package gomupdf

import (
	"math"
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

func approx(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func rectApprox(a, b geometry.Rect, eps float64) bool {
	return approx(a.X0, b.X0, eps) && approx(a.Y0, b.Y0, eps) &&
		approx(a.X1, b.X1, eps) && approx(a.Y1, b.Y1, eps)
}

// A freshly created A4-ish page reports its MediaBox and a zero default rotation.
func TestMediaBoxAndDefaultRotation(t *testing.T) {
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.NewPage(595, 842); err != nil {
		t.Fatal(err)
	}
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatal(err)
	}

	mb, err := p.MediaBox()
	if err != nil {
		t.Fatal(err)
	}
	want := geometry.Rect{X0: 0, Y0: 0, X1: 595, Y1: 842}
	if !rectApprox(mb, want, 0.5) {
		t.Errorf("MediaBox = %+v, want %+v", mb, want)
	}

	rot, err := p.Rotation()
	if err != nil {
		t.Fatal(err)
	}
	if rot != 0 {
		t.Errorf("default Rotation = %d, want 0", rot)
	}
}

// CropBox defaults to the MediaBox when the page declares none.
func TestCropBoxDefaultsToMediaBox(t *testing.T) {
	d, _ := NewPDF()
	defer d.Close()
	_ = d.NewPage(400, 300)
	p, _ := d.LoadPage(0)

	mb, err := p.MediaBox()
	if err != nil {
		t.Fatal(err)
	}
	cb, err := p.CropBox()
	if err != nil {
		t.Fatal(err)
	}
	if !rectApprox(mb, cb, 0.5) {
		t.Errorf("CropBox = %+v, want it to default to MediaBox %+v", cb, mb)
	}
}

// SetRotation persists across a save/reopen round-trip and normalizes the angle.
func TestSetRotationRoundTrip(t *testing.T) {
	cases := []struct{ in, want int }{
		{90, 90}, {180, 180}, {270, 270}, {360, 0}, {450, 90}, {-90, 270},
	}
	for _, c := range cases {
		d, _ := NewPDF()
		_ = d.NewPage(200, 200)
		p, _ := d.LoadPage(0)
		if err := p.SetRotation(c.in); err != nil {
			d.Close()
			t.Fatalf("SetRotation(%d): %v", c.in, err)
		}
		data, err := d.SaveBytes(true)
		d.Close()
		if err != nil {
			t.Fatalf("save: %v", err)
		}
		d2, err := OpenStream(data)
		if err != nil {
			t.Fatalf("reopen: %v", err)
		}
		p2, _ := d2.LoadPage(0)
		got, err := p2.Rotation()
		d2.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got != c.want {
			t.Errorf("SetRotation(%d) round-trip = %d, want %d", c.in, got, c.want)
		}
	}
}

// SetRotation rejects non-multiples of 90.
func TestSetRotationInvalid(t *testing.T) {
	d, _ := NewPDF()
	defer d.Close()
	_ = d.NewPage(200, 200)
	p, _ := d.LoadPage(0)
	if err := p.SetRotation(45); err == nil {
		t.Error("SetRotation(45) should error (not a multiple of 90)")
	}
}

// SetMediaBox and SetCropBox persist across a save/reopen round-trip.
func TestSetBoxesRoundTrip(t *testing.T) {
	d, _ := NewPDF()
	_ = d.NewPage(600, 600)
	p, _ := d.LoadPage(0)

	mb := geometry.Rect{X0: 0, Y0: 0, X1: 500, Y1: 700}
	if err := p.SetMediaBox(mb); err != nil {
		d.Close()
		t.Fatal(err)
	}
	cb := geometry.Rect{X0: 10, Y0: 20, X1: 400, Y1: 600}
	if err := p.SetCropBox(cb); err != nil {
		d.Close()
		t.Fatal(err)
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatal(err)
	}

	d2, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	p2, _ := d2.LoadPage(0)

	gotMB, _ := p2.MediaBox()
	if !rectApprox(gotMB, mb, 0.5) {
		t.Errorf("MediaBox round-trip = %+v, want %+v", gotMB, mb)
	}
	gotCB, _ := p2.CropBox()
	if !rectApprox(gotCB, cb, 0.5) {
		t.Errorf("CropBox round-trip = %+v, want %+v", gotCB, cb)
	}
}

// Reading geometry from a real fixture returns a sane, non-empty MediaBox.
func TestMediaBoxFixture(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	mb, err := p.MediaBox()
	if err != nil {
		t.Fatal(err)
	}
	if mb.Width() <= 0 || mb.Height() <= 0 {
		t.Errorf("fixture MediaBox is empty: %+v", mb)
	}
}

// Operating on a closed document is a clean error, not a crash.
func TestPageBoxClosedDoc(t *testing.T) {
	d, _ := NewPDF()
	_ = d.NewPage(100, 100)
	p, _ := d.LoadPage(0)
	d.Close()
	if _, err := p.MediaBox(); err == nil {
		t.Error("MediaBox on closed doc should error")
	}
	if _, err := p.Rotation(); err == nil {
		t.Error("Rotation on closed doc should error")
	}
}
