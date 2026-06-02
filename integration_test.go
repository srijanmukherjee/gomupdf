package gomupdf

import (
	"os"
	"strings"
	"testing"
)

// Integration tests against a real PDF. Gated on env vars so no PDF is committed:
//
//	GOMUPDF_TEST_PDF=/path/to.pdf GOMUPDF_TEST_PW=password go test ./...
//
// These exercise the public API against a user-supplied file rather than the
// committed fixtures.
func testDoc(t *testing.T) *Document {
	t.Helper()
	path := os.Getenv("GOMUPDF_TEST_PDF")
	if path == "" {
		t.Skip("set GOMUPDF_TEST_PDF (and GOMUPDF_TEST_PW) to run integration tests")
	}
	d, err := OpenWithPassword(path, os.Getenv("GOMUPDF_TEST_PW"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return d
}

func TestOpenAndPages(t *testing.T) {
	d := testDoc(t)
	defer d.Close()
	if n := d.PageCount(); n <= 0 {
		t.Fatalf("page count = %d", n)
	}
	if !d.IsEncrypted() {
		t.Log("note: test PDF is not encrypted")
	}
	if d.NeedsPass() {
		t.Error("NeedsPass should be false after successful auth")
	}
	p, _ := d.LoadPage(0)
	b, err := p.Bound()
	if err != nil {
		t.Fatalf("Bound: %v", err)
	}
	if b.Width() <= 0 || b.Height() <= 0 {
		t.Errorf("page 0 bound degenerate: %+v", b)
	}
}

func TestTextAndWords(t *testing.T) {
	d := testDoc(t)
	defer d.Close()
	full, err := d.Text()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(full) == "" {
		t.Fatal("empty document text")
	}
	p, _ := d.LoadPage(0)
	words, err := p.Words()
	if err != nil {
		t.Fatal(err)
	}
	if len(words) == 0 {
		t.Fatal("no words on page 0")
	}
	for _, w := range words {
		if w.BBox.W <= 0 || w.BBox.H <= 0 {
			t.Errorf("word %q has degenerate bbox %+v", w.Text, w.BBox)
			break
		}
	}
}

func TestSearch(t *testing.T) {
	d := testDoc(t)
	defer d.Close()
	needle := os.Getenv("GOMUPDF_TEST_NEEDLE")
	if needle == "" {
		// fall back to a word actually present on page 0
		p, _ := d.LoadPage(0)
		ws, _ := p.Words()
		if len(ws) == 0 {
			t.Skip("no words to search")
		}
		needle = ws[0].Text
	}
	found := 0
	for _, page := range d.Pages() {
		hits, err := page.Search(needle)
		if err != nil {
			t.Fatal(err)
		}
		for _, q := range hits {
			r := q.Rect()
			if r.Width() <= 0 || r.Height() <= 0 {
				t.Errorf("hit quad has degenerate rect %+v", r)
			}
		}
		found += len(hits)
	}
	if found == 0 {
		t.Errorf("expected at least one hit for %q", needle)
	}
}

func TestMetadata(t *testing.T) {
	d := testDoc(t)
	defer d.Close()
	meta, err := d.Metadata()
	if err != nil {
		t.Fatal(err)
	}
	if meta["format"] == "" {
		t.Error("expected a 'format' metadata value")
	}
}

func TestTOCAndLinks(t *testing.T) {
	d := testDoc(t)
	defer d.Close()
	toc, err := d.TOC()
	if err != nil {
		t.Fatalf("TOC: %v", err)
	}
	for _, e := range toc {
		if e.Level < 1 {
			t.Errorf("bad toc level %d", e.Level)
		}
	}
	for _, page := range d.Pages() {
		links, err := page.Links()
		if err != nil {
			t.Fatalf("Links: %v", err)
		}
		for _, l := range links {
			if l.Rect.Width() < 0 || l.Rect.Height() < 0 {
				t.Errorf("link has inverted rect %+v", l.Rect)
			}
		}
	}
}

func TestFindTablesNoError(t *testing.T) {
	d := testDoc(t)
	defer d.Close()
	for _, page := range d.Pages() {
		if _, err := page.FindTables(); err != nil {
			t.Fatalf("FindTables: %v", err)
		}
	}
}
