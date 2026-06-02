package gomupdf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildEncryptedPDF creates a small in-memory PDF with text, encrypted with the
// given user/owner passwords.
func buildEncryptedPDF(t *testing.T, userPwd, ownerPwd string) []byte {
	t.Helper()
	doc, err := NewPDF()
	if err != nil {
		t.Fatalf("NewPDF: %v", err)
	}
	defer doc.Close()
	if err := doc.NewPage(300, 300); err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	p, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}
	if err := p.InsertText(50, 250, 14, "Encrypted hello"); err != nil {
		t.Fatalf("InsertText: %v", err)
	}
	data, err := doc.SaveEncryptedBytes(userPwd, ownerPwd)
	if err != nil {
		t.Fatalf("SaveEncryptedBytes: %v", err)
	}
	return data
}

func TestEncryptionFlows(t *testing.T) {
	data := buildEncryptedPDF(t, "user", "owner")

	// OpenStream leaves the doc locked and flags it encrypted.
	d, err := OpenStream(data)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	if !d.NeedsPass() {
		t.Error("NeedsPass() = false, want true for encrypted doc")
	}
	if !d.IsEncrypted() {
		t.Error("IsEncrypted() = false, want true")
	}
	if d.Authenticate("wrong") {
		t.Error("Authenticate(wrong) = true, want false")
	}
	if !d.Authenticate("user") {
		t.Error("Authenticate(user) = false, want true")
	}
	if d.NeedsPass() {
		t.Error("NeedsPass() after auth = true, want false")
	}
	if !d.IsEncrypted() {
		t.Error("IsEncrypted() should stay true after auth")
	}
	d.Close()

	// OpenStreamWithPassword: correct password succeeds.
	ok, err := OpenStreamWithPassword(data, "user")
	if err != nil {
		t.Fatalf("OpenStreamWithPassword(user): %v", err)
	}
	if ok.NeedsPass() {
		t.Error("authenticated doc should not need a pass")
	}
	ok.Close()

	// OpenStreamWithPassword: wrong password errors.
	if _, err := OpenStreamWithPassword(data, "wrong"); err == nil {
		t.Error("OpenStreamWithPassword(wrong) = nil error, want error")
	}

	// OpenStreamWithPassword on empty input errors (OpenStream path).
	if _, err := OpenStreamWithPassword(nil, "user"); err == nil {
		t.Error("OpenStreamWithPassword(nil) = nil error, want error")
	}
}

func TestOpenWithPasswordFile(t *testing.T) {
	data := buildEncryptedPDF(t, "filepw", "")
	path := filepath.Join(t.TempDir(), "enc.pdf")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Correct password.
	d, err := OpenWithPassword(path, "filepw")
	if err != nil {
		t.Fatalf("OpenWithPassword: %v", err)
	}
	if d.NeedsPass() {
		t.Error("doc should be unlocked after OpenWithPassword")
	}
	d.Close()

	// Wrong password errors.
	if _, err := OpenWithPassword(path, "nope"); err == nil {
		t.Error("OpenWithPassword(wrong) = nil error, want error")
	}

	// Missing file errors.
	if _, err := OpenWithPassword(filepath.Join(t.TempDir(), "missing.pdf"), "x"); err == nil {
		t.Error("OpenWithPassword(missing) = nil error, want error")
	}
}

func TestOpenErrors(t *testing.T) {
	if _, err := OpenStream([]byte("not a pdf")); err == nil {
		t.Error("OpenStream(garbage) = nil error, want error")
	}
	if _, err := OpenStream(nil); err == nil {
		t.Error("OpenStream(nil) = nil error, want error")
	}
	if _, err := Open(filepath.Join(t.TempDir(), "does-not-exist.pdf")); err == nil {
		t.Error("Open(missing) = nil error, want error")
	}
}

func TestPagesIteratorAndCount(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()

	n := d.PageCount()
	if n <= 0 {
		t.Fatalf("PageCount() = %d, want > 0", n)
	}

	seen := 0
	for i, p := range d.Pages() {
		if p == nil {
			t.Fatal("nil page from iterator")
		}
		if p.Number != i {
			t.Errorf("page.Number = %d, want %d", p.Number, i)
		}
		seen++
	}
	if seen != n {
		t.Errorf("iterated %d pages, want %d", seen, n)
	}

	// Early break exercises the !yield return path.
	count := 0
	for range d.Pages() {
		count++
		break
	}
	if count != 1 {
		t.Errorf("early break iterated %d, want 1", count)
	}

	// LoadPage out of range.
	if _, err := d.LoadPage(-1); err == nil {
		t.Error("LoadPage(-1) = nil error, want error")
	}
	if _, err := d.LoadPage(n); err == nil {
		t.Error("LoadPage(n) = nil error, want error")
	}
}

func TestTextAggregations(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()

	full, err := d.Text()
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if !strings.Contains(full, "Boiling Points") {
		t.Errorf("Text() missing expected content; got %q", full)
	}

	byPage, err := d.TextByPage()
	if err != nil {
		t.Fatalf("TextByPage: %v", err)
	}
	if len(byPage) != d.PageCount() {
		t.Errorf("TextByPage len = %d, want %d", len(byPage), d.PageCount())
	}

	allLines, err := d.AllLines()
	if err != nil {
		t.Fatalf("AllLines: %v", err)
	}
	if len(allLines) == 0 {
		t.Error("AllLines() empty")
	}

	p, _ := d.LoadPage(0)
	pageText, err := p.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if !strings.Contains(pageText, "Boiling Points") {
		t.Errorf("GetText() missing expected content")
	}

	lines, err := p.Lines()
	if err != nil {
		t.Fatalf("Lines: %v", err)
	}
	if len(lines) == 0 {
		t.Error("Lines() empty")
	}
}

func TestStructuredTextSpansWords(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	js, err := p.StructuredJSON()
	if err != nil {
		t.Fatalf("StructuredJSON: %v", err)
	}
	if !strings.Contains(js, "blocks") {
		t.Errorf("StructuredJSON missing 'blocks'; got %.80q", js)
	}

	blocks, err := p.StructuredText()
	if err != nil {
		t.Fatalf("StructuredText: %v", err)
	}
	if len(blocks) == 0 {
		t.Fatal("StructuredText returned no blocks")
	}

	spans, err := p.Spans()
	if err != nil {
		t.Fatalf("Spans: %v", err)
	}
	if len(spans) == 0 {
		t.Fatal("Spans returned none")
	}

	words, err := p.Words()
	if err != nil {
		t.Fatalf("Words: %v", err)
	}
	if len(words) == 0 {
		t.Fatal("Words returned none")
	}
	// Exercise the Rect value-type accessors on a real word box.
	w := words[0]
	if w.BBox.X1() < w.BBox.X {
		t.Errorf("X1 %v < X %v", w.BBox.X1(), w.BBox.X)
	}
	if w.BBox.Y1() < w.BBox.Y {
		t.Errorf("Y1 %v < Y %v", w.BBox.Y1(), w.BBox.Y)
	}
	cy := w.BBox.CenterY()
	if cy < w.BBox.Y || cy > w.BBox.Y1() {
		t.Errorf("CenterY %v out of [%v,%v]", cy, w.BBox.Y, w.BBox.Y1())
	}
}

func TestSearchFixture(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	quads, err := p.Search("Boiling")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(quads) == 0 {
		t.Fatal("Search('Boiling') found nothing")
	}

	rects, err := p.SearchRects("Boiling")
	if err != nil {
		t.Fatalf("SearchRects: %v", err)
	}
	if len(rects) != len(quads) {
		t.Errorf("SearchRects len = %d, want %d", len(rects), len(quads))
	}
	if rects[0].Width() <= 0 || rects[0].Height() <= 0 {
		t.Errorf("search rect degenerate: %+v", rects[0])
	}

	// Empty needle short-circuits to nil.
	empty, err := p.Search("")
	if err != nil || empty != nil {
		t.Errorf("Search(\"\") = %v, %v; want nil, nil", empty, err)
	}

	// A needle that does not exist returns no hits, no error.
	none, err := p.Search("zzz-not-present-zzz")
	if err != nil {
		t.Fatalf("Search(absent): %v", err)
	}
	if len(none) != 0 {
		t.Errorf("Search(absent) = %d hits, want 0", len(none))
	}
}

func TestMetadataFixture(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	meta, err := d.Metadata()
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if meta == nil {
		t.Fatal("Metadata() returned nil map")
	}
	if _, ok := meta["format"]; !ok {
		t.Errorf("Metadata missing 'format' key; got %v", meta)
	}
}

func TestTOCAndLinksFixture(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()

	// TOC may be empty for these fixtures; covering the call with no error is fine.
	if _, err := d.TOC(); err != nil {
		t.Errorf("TOC: %v", err)
	}

	p, _ := d.LoadPage(0)
	if _, err := p.Links(); err != nil {
		t.Errorf("Links: %v", err)
	}
}

func TestSavePNGAndSave(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	pm, err := p.Pixmap()
	if err != nil {
		t.Fatalf("Pixmap: %v", err)
	}
	pngPath := filepath.Join(t.TempDir(), "out.png")
	if err := pm.SavePNG(pngPath); err != nil {
		t.Fatalf("SavePNG: %v", err)
	}
	if _, err := pm.Image(); err != nil {
		t.Fatalf("Image: %v", err)
	}

	// Document.Save to a temp file.
	pdfPath := filepath.Join(t.TempDir(), "out.pdf")
	if err := d.Save(pdfPath, true); err != nil {
		t.Fatalf("Save: %v", err)
	}
	re, err := Open(pdfPath)
	if err != nil {
		t.Fatalf("reopen saved: %v", err)
	}
	re.Close()
}

func TestFindTablesTextStrategy(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	// Text strategy derives rulings from word alignment, exercising
	// wordsToEdgesV/H and clusterObjects.
	tabs, err := p.FindTables(TableSettings{Strategy: StrategyText})
	if err != nil {
		t.Fatalf("FindTables(text): %v", err)
	}
	for _, tab := range tabs {
		if tab.NumRows() < 0 || tab.NumCols() < 0 {
			t.Errorf("degenerate table dims %dx%d", tab.NumRows(), tab.NumCols())
		}
	}

	// Default settings (no opts) also runs the text strategy.
	if _, err := p.FindTables(); err != nil {
		t.Fatalf("FindTables(default): %v", err)
	}

	// Empty-strategy string falls back to text.
	if _, err := p.FindTables(TableSettings{Strategy: ""}); err != nil {
		t.Fatalf("FindTables(empty strategy): %v", err)
	}
}

func TestPixmapGrayPNGAndImage(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)

	// Grayscale -> Image (Gray path) -> PNG.
	gray, err := p.Pixmap(PixmapOptions{Gray: true})
	if err != nil {
		t.Fatalf("Pixmap(gray): %v", err)
	}
	if _, err := gray.Image(); err != nil {
		t.Fatalf("gray Image: %v", err)
	}
	if _, err := gray.PNG(); err != nil {
		t.Fatalf("gray PNG: %v", err)
	}

	// Gray + alpha -> Image (N==2 path).
	ga, err := p.Pixmap(PixmapOptions{Gray: true, Alpha: true})
	if err != nil {
		t.Fatalf("Pixmap(gray+alpha): %v", err)
	}
	if ga.N != 2 {
		t.Logf("gray+alpha N = %d (expected 2)", ga.N)
	}
	if _, err := ga.Image(); err != nil {
		t.Fatalf("gray+alpha Image: %v", err)
	}

	// RGBA -> Image (N==4 path).
	rgba, _ := p.Pixmap(PixmapOptions{Alpha: true})
	if _, err := rgba.Image(); err != nil {
		t.Fatalf("rgba Image: %v", err)
	}

	// Out-of-range Pixel returns nil.
	if px := gray.Pixel(-1, 0); px != nil {
		t.Error("Pixel(-1,0) should be nil")
	}
	if px := gray.Pixel(gray.Width, 0); px != nil {
		t.Error("Pixel(width,0) should be nil")
	}
}

func TestSaveClosedDocErrors(t *testing.T) {
	d, _ := NewPDF()
	d.Close()
	if _, err := d.SaveBytes(true); err == nil {
		t.Error("SaveBytes on closed doc = nil error, want error")
	}
	if _, err := d.Metadata(); err == nil {
		t.Error("Metadata on closed doc = nil error, want error")
	}
	if err := d.Save(filepath.Join(t.TempDir(), "x.pdf"), false); err == nil {
		t.Error("Save on closed doc = nil error, want error")
	}
}
