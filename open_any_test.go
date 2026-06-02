package gomupdf

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// makePNG builds a small opaque PNG and returns its encoded bytes.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// An image opens as a single-page document with sensible page bounds.
func TestOpenImageStream(t *testing.T) {
	data := makePNG(t, 64, 48)
	d, err := OpenAnyStream(data, "png")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if d.PageCount() != 1 {
		t.Fatalf("page count = %d, want 1", d.PageCount())
	}
	p, _ := d.LoadPage(0)
	b, err := p.Bound()
	if err != nil {
		t.Fatal(err)
	}
	if b.Width() <= 0 || b.Height() <= 0 {
		t.Errorf("image page bounds empty: %+v", b)
	}
}

// OpenAny infers the format from the file extension.
func TestOpenAnyFromExtension(t *testing.T) {
	data := makePNG(t, 32, 32)
	path := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := OpenAny(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if d.PageCount() != 1 {
		t.Errorf("page count = %d, want 1", d.PageCount())
	}
}

// An empty filetype hint is treated as PDF, matching OpenStream.
func TestOpenAnyDefaultsToPDF(t *testing.T) {
	src := buildDoc(t)
	pdf, _ := src.SaveBytes(true)
	src.Close()

	d, err := OpenAnyStream(pdf, "")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if d.PageCount() != 3 {
		t.Errorf("page count = %d, want 3", d.PageCount())
	}
}

// Converting an image document yields a writable single-page PDF that reopens.
func TestConvertImageToPDF(t *testing.T) {
	data := makePNG(t, 80, 60)
	src, err := OpenAnyStream(data, "png")
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	pdf, err := src.ConvertToPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer pdf.Close()
	if pdf.PageCount() != 1 {
		t.Fatalf("converted page count = %d, want 1", pdf.PageCount())
	}
	// It is a real PDF: it serializes and reopens.
	out, err := pdf.SaveBytes(true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Error("converted output is not a PDF")
	}
	rd, err := OpenStream(out)
	if err != nil {
		t.Fatal(err)
	}
	rd.Close()
}

// Converting a PDF preserves its page count and text.
func TestConvertPDFPreservesContent(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	wantPages := d.PageCount()

	conv, err := d.ConvertToPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer conv.Close()
	if conv.PageCount() != wantPages {
		t.Errorf("converted page count = %d, want %d", conv.PageCount(), wantPages)
	}
	txt, err := conv.Text()
	if err != nil {
		t.Fatal(err)
	}
	if len(txt) == 0 {
		t.Error("converted PDF lost all text")
	}
}

// Errors are clean for empty input and closed documents.
func TestOpenAnyErrors(t *testing.T) {
	if _, err := OpenAnyStream(nil, "png"); err == nil {
		t.Error("empty input should error")
	}
	d := buildDoc(t)
	d.Close()
	if _, err := d.ConvertToPDF(); err == nil {
		t.Error("ConvertToPDF on closed doc should error")
	}
}
