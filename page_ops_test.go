package gomupdf

import (
	"fmt"
	"strings"
	"testing"
)

// makeLabeled creates a new PDF with n pages where page i contains the text
// "PAGE{i}". Returns the document (caller must Close).
func makeLabeled(t *testing.T, n int) *Document {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatalf("NewPDF: %v", err)
	}
	for i := 0; i < n; i++ {
		if err := d.NewPage(200, 200); err != nil {
			d.Close()
			t.Fatalf("NewPage(%d): %v", i, err)
		}
		p, err := d.LoadPage(i)
		if err != nil {
			d.Close()
			t.Fatalf("LoadPage(%d): %v", i, err)
		}
		if err := p.InsertText(20, 100, 20, fmt.Sprintf("PAGE%d", i)); err != nil {
			d.Close()
			t.Fatalf("InsertText(page %d): %v", i, err)
		}
	}
	return d
}

// pageLabel returns the trimmed text of page i in d, failing the test on error.
func pageLabel(t *testing.T, d *Document, i int) string {
	t.Helper()
	p, err := d.LoadPage(i)
	if err != nil {
		t.Fatalf("LoadPage(%d): %v", i, err)
	}
	txt, err := p.GetText()
	if err != nil {
		t.Fatalf("GetText(page %d): %v", i, err)
	}
	return strings.TrimSpace(txt)
}

// containsLabel returns true if the text of page i contains label.
func containsLabel(t *testing.T, d *Document, i int, label string) bool {
	t.Helper()
	return strings.Contains(pageLabel(t, d, i), label)
}

// assertOrder checks that page i contains label, fataling on mismatch.
func assertOrder(t *testing.T, d *Document, i int, label string) {
	t.Helper()
	txt := pageLabel(t, d, i)
	if !strings.Contains(txt, label) {
		t.Errorf("page %d: got %q, want label %q", i, txt, label)
	}
}

// ── CopyPage ──────────────────────────────────────────────────────────────────

func TestCopyPageCount(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()

	// CopyPage(0, 3) on a 3-page doc → 4 pages; copy lands at index 3.
	if err := d.CopyPage(0, 3); err != nil {
		t.Fatalf("CopyPage(0,3): %v", err)
	}
	if d.PageCount() != 4 {
		t.Fatalf("PageCount = %d, want 4", d.PageCount())
	}
	assertOrder(t, d, 3, "PAGE0")
	// Original pages unchanged.
	assertOrder(t, d, 0, "PAGE0")
	assertOrder(t, d, 1, "PAGE1")
	assertOrder(t, d, 2, "PAGE2")
}

func TestCopyPageRoundTrip(t *testing.T) {
	d := makeLabeled(t, 3)
	if err := d.CopyPage(2, 0); err != nil {
		d.Close()
		t.Fatalf("CopyPage(2,0): %v", err)
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}
	d2, err := OpenStream(data)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer d2.Close()
	if d2.PageCount() != 4 {
		t.Fatalf("after round-trip PageCount = %d, want 4", d2.PageCount())
	}
	// Copy of page 2 inserted at index 0.
	assertOrder(t, d2, 0, "PAGE2")
}

func TestCopyPageOutOfRange(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	if err := d.CopyPage(99, 0); err == nil {
		t.Error("CopyPage(99,0) should error on out-of-range from")
	}
	if err := d.CopyPage(0, 99); err == nil {
		t.Error("CopyPage(0,99) should error on out-of-range to")
	}
}

// ── MovePage ─────────────────────────────────────────────────────────────────

// MovePage(0→2) on [PAGE0, PAGE1, PAGE2]:
// Delete page 0 → [PAGE1, PAGE2]; to(2) > from(0) → ins = 2-1 = 1.
// Insert PAGE0 at 1 → [PAGE1, PAGE0, PAGE2].
func TestMovePageForward(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	if err := d.MovePage(0, 2); err != nil {
		t.Fatalf("MovePage(0,2): %v", err)
	}
	if d.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3", d.PageCount())
	}
	assertOrder(t, d, 0, "PAGE1")
	assertOrder(t, d, 1, "PAGE0")
	assertOrder(t, d, 2, "PAGE2")
}

// MovePage(2→0) on [PAGE0, PAGE1, PAGE2] → [PAGE2, PAGE0, PAGE1]
func TestMovePageBackward(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	if err := d.MovePage(2, 0); err != nil {
		t.Fatalf("MovePage(2,0): %v", err)
	}
	assertOrder(t, d, 0, "PAGE2")
	assertOrder(t, d, 1, "PAGE0")
	assertOrder(t, d, 2, "PAGE1")
}

// MovePage with same index should be a no-op (or succeed cleanly).
func TestMovePageSameIndex(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	if err := d.MovePage(1, 1); err != nil {
		t.Fatalf("MovePage(1,1): %v", err)
	}
	assertOrder(t, d, 0, "PAGE0")
	assertOrder(t, d, 1, "PAGE1")
	assertOrder(t, d, 2, "PAGE2")
}

func TestMovePageOutOfRange(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	if err := d.MovePage(0, 99); err == nil {
		t.Error("MovePage(0,99) should error on out-of-range to")
	}
	if err := d.MovePage(99, 0); err == nil {
		t.Error("MovePage(99,0) should error on out-of-range from")
	}
}

func TestMovePageRoundTrip(t *testing.T) {
	d := makeLabeled(t, 4)
	// Move page 3 to index 1: [PAGE0, PAGE1, PAGE2, PAGE3] → [PAGE0, PAGE3, PAGE1, PAGE2]
	if err := d.MovePage(3, 1); err != nil {
		d.Close()
		t.Fatalf("MovePage(3,1): %v", err)
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}
	d2, err := OpenStream(data)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer d2.Close()
	assertOrder(t, d2, 0, "PAGE0")
	assertOrder(t, d2, 1, "PAGE3")
	assertOrder(t, d2, 2, "PAGE1")
	assertOrder(t, d2, 3, "PAGE2")
}

// ── SelectPages ──────────────────────────────────────────────────────────────

func TestSelectPagesSubset(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	// [2, 0] → 2 pages: page0="PAGE2", page1="PAGE0"
	if err := d.SelectPages([]int{2, 0}); err != nil {
		t.Fatalf("SelectPages([2,0]): %v", err)
	}
	if d.PageCount() != 2 {
		t.Fatalf("PageCount = %d, want 2", d.PageCount())
	}
	assertOrder(t, d, 0, "PAGE2")
	assertOrder(t, d, 1, "PAGE0")
}

func TestSelectPagesDuplicate(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	// [1, 1, 1] → 3 pages all "PAGE1"
	if err := d.SelectPages([]int{1, 1, 1}); err != nil {
		t.Fatalf("SelectPages([1,1,1]): %v", err)
	}
	if d.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3", d.PageCount())
	}
	for i := 0; i < 3; i++ {
		assertOrder(t, d, i, "PAGE1")
	}
}

func TestSelectPagesEmpty(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	if err := d.SelectPages(nil); err == nil {
		t.Error("SelectPages(nil) should error")
	}
	if err := d.SelectPages([]int{}); err == nil {
		t.Error("SelectPages([]) should error")
	}
}

func TestSelectPagesOutOfRange(t *testing.T) {
	d := makeLabeled(t, 3)
	defer d.Close()
	if err := d.SelectPages([]int{0, 99}); err == nil {
		t.Error("SelectPages([0,99]) should error on out-of-range index")
	}
}

func TestSelectPagesRoundTrip(t *testing.T) {
	d := makeLabeled(t, 5)
	// Reverse order: [4,3,2,1,0]
	if err := d.SelectPages([]int{4, 3, 2, 1, 0}); err != nil {
		d.Close()
		t.Fatalf("SelectPages reverse: %v", err)
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}
	d2, err := OpenStream(data)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer d2.Close()
	if d2.PageCount() != 5 {
		t.Fatalf("PageCount = %d, want 5", d2.PageCount())
	}
	for i := 0; i < 5; i++ {
		want := fmt.Sprintf("PAGE%d", 4-i)
		assertOrder(t, d2, i, want)
	}
}

// ── InsertPDFRange ────────────────────────────────────────────────────────────

func TestInsertPDFRange(t *testing.T) {
	// Build a 4-page source doc.
	src := makeLabeled(t, 4)
	srcBytes, err := src.SaveBytes(true)
	src.Close()
	if err != nil {
		t.Fatalf("SaveBytes (src): %v", err)
	}

	// Destination: 1 page.
	dst := makeLabeled(t, 1)
	defer dst.Close()
	// Insert source pages [1, 2] at end → dest gets 3 pages total.
	if err := dst.InsertPDFRange(srcBytes, 1, 2, ""); err != nil {
		t.Fatalf("InsertPDFRange: %v", err)
	}
	if dst.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3", dst.PageCount())
	}
	assertOrder(t, dst, 0, "PAGE0") // original dest page
	assertOrder(t, dst, 1, "PAGE1") // from source
	assertOrder(t, dst, 2, "PAGE2") // from source
}

func TestInsertPDFRangeRoundTrip(t *testing.T) {
	src := makeLabeled(t, 4)
	srcBytes, err := src.SaveBytes(true)
	src.Close()
	if err != nil {
		t.Fatalf("SaveBytes (src): %v", err)
	}

	dst := makeLabeled(t, 1)
	if err := dst.InsertPDFRange(srcBytes, 1, 2, ""); err != nil {
		dst.Close()
		t.Fatalf("InsertPDFRange: %v", err)
	}
	data, err := dst.SaveBytes(true)
	dst.Close()
	if err != nil {
		t.Fatalf("SaveBytes (dst): %v", err)
	}

	d2, err := OpenStream(data)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer d2.Close()
	if d2.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3", d2.PageCount())
	}
	assertOrder(t, d2, 1, "PAGE1")
	assertOrder(t, d2, 2, "PAGE2")
}

func TestInsertPDFRangeClamping(t *testing.T) {
	// fromPage/toPage beyond source range should be clamped, not error.
	src := makeLabeled(t, 3)
	srcBytes, err := src.SaveBytes(true)
	src.Close()
	if err != nil {
		t.Fatalf("SaveBytes (src): %v", err)
	}

	dst := makeLabeled(t, 1)
	defer dst.Close()
	// toPage=99 should clamp to 2 (last page).
	if err := dst.InsertPDFRange(srcBytes, 0, 99, ""); err != nil {
		t.Fatalf("InsertPDFRange with clamped toPage: %v", err)
	}
	if dst.PageCount() != 4 {
		t.Fatalf("PageCount = %d, want 4", dst.PageCount())
	}
}

func TestInsertPDFRangeEmptySrc(t *testing.T) {
	dst := makeLabeled(t, 1)
	defer dst.Close()
	if err := dst.InsertPDFRange(nil, 0, 0, ""); err == nil {
		t.Error("InsertPDFRange(nil) should error")
	}
	if err := dst.InsertPDFRange([]byte{}, 0, 0, ""); err == nil {
		t.Error("InsertPDFRange(empty) should error")
	}
}

// ── Closed-document errors ────────────────────────────────────────────────────

func TestPageOpsClosedDocument(t *testing.T) {
	d := makeLabeled(t, 3)
	d.Close()

	if err := d.CopyPage(0, 1); err == nil {
		t.Error("CopyPage on closed doc should error")
	}
	if err := d.MovePage(0, 1); err == nil {
		t.Error("MovePage on closed doc should error")
	}
	if err := d.SelectPages([]int{0}); err == nil {
		t.Error("SelectPages on closed doc should error")
	}

	src := makeLabeled(t, 1)
	srcBytes, _ := src.SaveBytes(true)
	src.Close()
	if err := d.InsertPDFRange(srcBytes, 0, 0, ""); err == nil {
		t.Error("InsertPDFRange on closed doc should error")
	}
}

// ── containsLabel used directly so linter doesn't complain ───────────────────

var _ = containsLabel
