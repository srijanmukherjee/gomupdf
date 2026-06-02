package gomupdf

import (
	"strings"
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// newRedactDoc creates a fresh PDF with one 400×400-point page containing the
// text "SECRET public" inserted at (50, 200) with size 20.
func newRedactDoc(t *testing.T) (*Document, *Page) {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatalf("NewPDF: %v", err)
	}
	if err := d.NewPage(400, 400); err != nil {
		d.Close()
		t.Fatalf("NewPage: %v", err)
	}
	p, err := d.LoadPage(0)
	if err != nil {
		d.Close()
		t.Fatalf("LoadPage: %v", err)
	}
	if err := p.InsertText(50, 200, 20, "SECRET public"); err != nil {
		d.Close()
		t.Fatalf("InsertText: %v", err)
	}
	return d, p
}

// redactRect is a rect that reliably covers the word "SECRET" in the test doc.
// InsertText draws at y=200 baseline, 20pt font, starting at x=50. The word
// "SECRET" is ~6 chars × ~12pt advance ≈ 72pt wide. We use a generous rect.
var secretRect = geometry.Rect{X0: 40, Y0: 178, X1: 160, Y1: 210}

// TestAddRedactionDoesNotRemoveText verifies that AddRedaction alone (without
// ApplyRedactions) leaves the text intact.
func TestAddRedactionDoesNotRemoveText(t *testing.T) {
	d, p := newRedactDoc(t)
	defer d.Close()

	if err := p.AddRedaction(secretRect, RedactOptions{}); err != nil {
		t.Fatalf("AddRedaction: %v", err)
	}

	txt, err := p.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if !strings.Contains(txt, "SECRET") {
		t.Errorf("text should still contain SECRET before apply; got %q", strings.TrimSpace(txt))
	}
}

// TestApplyRedactionsRemovesText is the core test: add a redaction covering
// "SECRET", apply it, save, reopen, and verify the text is gone.
func TestApplyRedactionsRemovesText(t *testing.T) {
	d, p := newRedactDoc(t)
	defer d.Close()

	// Confirm text is present before redaction.
	txt, err := p.GetText()
	if err != nil {
		t.Fatalf("GetText before: %v", err)
	}
	if !strings.Contains(txt, "SECRET") {
		t.Skipf("InsertText/GetText round-trip broken in this build; got %q", strings.TrimSpace(txt))
	}

	if err := p.AddRedaction(secretRect, RedactOptions{}); err != nil {
		t.Fatalf("AddRedaction: %v", err)
	}

	n, err := p.ApplyRedactions()
	if err != nil {
		t.Fatalf("ApplyRedactions: %v", err)
	}
	if n != 1 {
		t.Errorf("ApplyRedactions returned %d, want 1", n)
	}

	// Save and reopen to confirm the redaction is permanent.
	data, err := d.SaveBytes(true)
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}
	re, err := OpenStream(data)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer re.Close()

	rp, err := re.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}
	after, err := rp.GetText()
	if err != nil {
		t.Fatalf("GetText after: %v", err)
	}
	if strings.Contains(after, "SECRET") {
		t.Errorf("redacted text 'SECRET' still present after apply+save; got %q", strings.TrimSpace(after))
	}
}

// TestApplyRedactionsCount verifies the return value matches the number of
// redaction annotations added.
func TestApplyRedactionsCount(t *testing.T) {
	d, p := newRedactDoc(t)
	defer d.Close()

	// Add two redaction annotations (covering different parts of the text).
	rects := []geometry.Rect{
		{X0: 40, Y0: 178, X1: 160, Y1: 210},  // covers SECRET
		{X0: 160, Y0: 178, X1: 300, Y1: 210}, // covers public
	}
	for _, r := range rects {
		if err := p.AddRedaction(r, RedactOptions{}); err != nil {
			t.Fatalf("AddRedaction: %v", err)
		}
	}

	n, err := p.ApplyRedactions()
	if err != nil {
		t.Fatalf("ApplyRedactions: %v", err)
	}
	if n != 2 {
		t.Errorf("ApplyRedactions returned %d, want 2", n)
	}
}

// TestApplyRedactionsZero verifies that applying with no redaction annotations
// returns 0 and no error.
func TestApplyRedactionsZero(t *testing.T) {
	d, p := newRedactDoc(t)
	defer d.Close()

	n, err := p.ApplyRedactions()
	if err != nil {
		t.Fatalf("ApplyRedactions on clean page: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 redactions, got %d", n)
	}
}

// TestRedactionClosedDocument verifies that both methods return an error when
// called on a closed document.
func TestRedactionClosedDocument(t *testing.T) {
	d, p := newRedactDoc(t)
	d.Close()

	if err := p.AddRedaction(secretRect, RedactOptions{}); err == nil {
		t.Error("AddRedaction: expected error on closed document, got nil")
	}
	if _, err := p.ApplyRedactions(); err == nil {
		t.Error("ApplyRedactions: expected error on closed document, got nil")
	}
}

// TestAddRedactionWithFill verifies that a custom fill colour is accepted
// without error (visual output not verified in unit tests).
func TestAddRedactionWithFill(t *testing.T) {
	d, p := newRedactDoc(t)
	defer d.Close()

	blue := [3]float64{0, 0, 1}
	if err := p.AddRedaction(secretRect, RedactOptions{Fill: &blue}); err != nil {
		t.Fatalf("AddRedaction with fill: %v", err)
	}
	n, err := p.ApplyRedactions()
	if err != nil {
		t.Fatalf("ApplyRedactions: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
}
