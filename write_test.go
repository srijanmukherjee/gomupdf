package gomupdf

import (
	"os"
	"strings"
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

func TestNewPDFSaveRoundTrip(t *testing.T) {
	doc, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	if err := doc.NewPage(595, 842); err != nil {
		t.Fatal(err)
	}
	if err := doc.NewPage(595, 842); err != nil {
		t.Fatal(err)
	}
	p0, _ := doc.LoadPage(0)
	if err := p0.InsertText(72, 800, 12, "Hello gomupdf"); err != nil {
		t.Fatal(err)
	}
	data, err := doc.SaveBytes(true)
	if err != nil {
		t.Fatal(err)
	}

	// Reopen and verify.
	re, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	defer re.Close()
	if re.PageCount() != 2 {
		t.Fatalf("page count = %d, want 2", re.PageCount())
	}
	rp0, _ := re.LoadPage(0)
	txt, _ := rp0.GetText()
	if !strings.Contains(txt, "Hello gomupdf") {
		t.Errorf("inserted text not found; got %q", strings.TrimSpace(txt))
	}
	b, _ := rp0.Bound()
	if int(b.Width()) != 595 || int(b.Height()) != 842 {
		t.Errorf("page size = %vx%v, want 595x842", b.Width(), b.Height())
	}
}

func TestDeletePage(t *testing.T) {
	doc, _ := NewPDF()
	defer doc.Close()
	for i := 0; i < 3; i++ {
		if err := doc.NewPage(200, 200); err != nil {
			t.Fatal(err)
		}
	}
	if err := doc.DeletePage(1); err != nil {
		t.Fatal(err)
	}
	data, _ := doc.SaveBytes(true)
	re, _ := OpenStream(data)
	defer re.Close()
	if re.PageCount() != 2 {
		t.Fatalf("after delete, page count = %d, want 2", re.PageCount())
	}
}

func TestInsertPDF(t *testing.T) {
	src, err := os.ReadFile(res + "image-file1.pdf")
	if err != nil {
		t.Fatal(err)
	}
	srcDoc, _ := OpenStream(src)
	srcPages := srcDoc.PageCount()
	srcDoc.Close()

	doc, _ := NewPDF()
	defer doc.Close()
	doc.NewPage(300, 300)
	if err := doc.InsertPDF(src, ""); err != nil {
		t.Fatal(err)
	}
	data, _ := doc.SaveBytes(true)
	re, _ := OpenStream(data)
	defer re.Close()
	if re.PageCount() != 1+srcPages {
		t.Fatalf("merged page count = %d, want %d", re.PageCount(), 1+srcPages)
	}
}

func TestInsertImage(t *testing.T) {
	// Extract a JPEG from a fixture, then place it on a new page.
	srcDoc := openFixture(t, "image-file1.pdf")
	sp, _ := srcDoc.LoadPage(0)
	ex, err := sp.ExtractImage(0)
	srcDoc.Close()
	if err != nil || ex == nil {
		t.Fatalf("extract source image: %v", err)
	}

	doc, _ := NewPDF()
	defer doc.Close()
	doc.NewPage(400, 400)
	p, _ := doc.LoadPage(0)
	if err := p.InsertImage(geometry.NewRect(50, 50, 350, 350), ex.Bytes); err != nil {
		t.Fatal(err)
	}
	data, err := doc.SaveBytes(true)
	if err != nil {
		t.Fatal(err)
	}
	re, _ := OpenStream(data)
	defer re.Close()
	rp, _ := re.LoadPage(0)
	imgs, err := rp.GetImages()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 1 {
		t.Fatalf("inserted image not found; got %d images", len(imgs))
	}
	if imgs[0].BBox.Width() <= 0 || imgs[0].BBox.Height() <= 0 {
		t.Errorf("placed image has degenerate bbox %+v", imgs[0].BBox)
	}
}

func TestSaveEncrypted(t *testing.T) {
	doc, _ := NewPDF()
	defer doc.Close()
	doc.NewPage(300, 300)
	p, _ := doc.LoadPage(0)
	p.InsertText(50, 250, 14, "secret content")
	data, err := doc.SaveEncryptedBytes("opensesame", "")
	if err != nil {
		t.Fatal(err)
	}
	re, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	defer re.Close()
	if !re.NeedsPass() {
		t.Fatal("re-opened doc should require a password")
	}
	if re.Authenticate("wrong") {
		t.Error("wrong password should fail")
	}
	if !re.Authenticate("opensesame") {
		t.Fatal("correct password should unlock")
	}
	rp, _ := re.LoadPage(0)
	txt, _ := rp.GetText()
	if !strings.Contains(txt, "secret content") {
		t.Errorf("decrypted text missing; got %q", strings.TrimSpace(txt))
	}
}

func TestAddRectAnnot(t *testing.T) {
	doc, _ := NewPDF()
	defer doc.Close()
	doc.NewPage(400, 400)
	p, _ := doc.LoadPage(0)
	if err := p.AddRectAnnot(geometry.NewRect(50, 50, 200, 150)); err != nil {
		t.Fatal(err)
	}
	data, err := doc.SaveBytes(true)
	if err != nil {
		t.Fatal(err)
	}
	re, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	defer re.Close()
	if re.PageCount() != 1 {
		t.Errorf("page count = %d, want 1", re.PageCount())
	}
}
