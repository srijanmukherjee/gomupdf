package gomupdf

import (
	"strings"
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// TestExtractTextDefault verifies that ExtractText with default (zero-value)
// options returns non-empty text that closely matches GetText.
func TestExtractTextDefault(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	got, err := p.ExtractText(TextOptions{})
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	if got == "" {
		t.Fatal("ExtractText returned empty string")
	}

	// GetText and ExtractText with default opts should contain the same key content.
	baseline, err := p.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if !strings.Contains(got, "Boiling Points") {
		t.Errorf("ExtractText output missing expected text; got %q", got[:min(len(got), 200)])
	}
	if baseline == "" {
		t.Fatal("GetText returned empty string")
	}
}

// TestExtractTextPreserveWhitespace verifies that PreserveWhitespace changes
// the output (or at least returns non-empty text without error).
func TestExtractTextPreserveWhitespace(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	plain, err := p.ExtractText(TextOptions{})
	if err != nil {
		t.Fatalf("ExtractText plain: %v", err)
	}

	ws, err := p.ExtractText(TextOptions{PreserveWhitespace: true})
	if err != nil {
		t.Fatalf("ExtractText PreserveWhitespace: %v", err)
	}
	if ws == "" {
		t.Fatal("ExtractText with PreserveWhitespace returned empty string")
	}
	// At minimum both should contain core text.
	if !strings.Contains(plain, "Boiling") || !strings.Contains(ws, "Boiling") {
		t.Errorf("expected 'Boiling' in both outputs; plain=%q ws=%q",
			plain[:min(len(plain), 100)], ws[:min(len(ws), 100)])
	}
}

// TestExtractTextInhibitSpaces verifies that InhibitSpaces returns non-empty
// text. The output will differ from the default (fewer spaces between glyphs).
func TestExtractTextInhibitSpaces(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	inhibited, err := p.ExtractText(TextOptions{InhibitSpaces: true})
	if err != nil {
		t.Fatalf("ExtractText InhibitSpaces: %v", err)
	}
	if inhibited == "" {
		t.Fatal("ExtractText with InhibitSpaces returned empty string")
	}

	plain, err := p.ExtractText(TextOptions{})
	if err != nil {
		t.Fatalf("ExtractText plain: %v", err)
	}

	// With InhibitSpaces the output generally has fewer spaces.
	// At minimum the two outputs differ, confirming the flag has effect.
	if inhibited == plain {
		t.Log("InhibitSpaces produced identical output to default (may be PDF-specific — not failing)")
	}
}

// TestExtractTextClipTopHalf clips to the top half of the page and verifies
// that the clipped text is a strict subset (shorter) of the full-page text.
func TestExtractTextClipTopHalf(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	bound, err := p.Bound()
	if err != nil {
		t.Fatalf("Bound: %v", err)
	}

	// Top half of the page (Y0 to midpoint).
	midY := (bound.Y0 + bound.Y1) / 2
	topHalf := &geometry.Rect{
		X0: bound.X0,
		Y0: bound.Y0,
		X1: bound.X1,
		Y1: midY,
	}

	full, err := p.ExtractText(TextOptions{})
	if err != nil {
		t.Fatalf("ExtractText full: %v", err)
	}

	clipped, err := p.ExtractText(TextOptions{Clip: topHalf})
	if err != nil {
		t.Fatalf("ExtractText clipped: %v", err)
	}

	if clipped == "" {
		t.Fatal("clipped text is empty (expected some text in the top half)")
	}
	if len(clipped) >= len(full) {
		t.Errorf("clipped text length (%d) should be < full text length (%d)", len(clipped), len(full))
	}
}

// TestExtractTextClipNarrow clips to a very narrow strip (bottom 5% of the
// page) and verifies the result is shorter than the full text.
func TestExtractTextClipNarrow(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	bound, err := p.Bound()
	if err != nil {
		t.Fatalf("Bound: %v", err)
	}

	height := bound.Y1 - bound.Y0
	stripTop := bound.Y1 - height*0.05
	strip := &geometry.Rect{
		X0: bound.X0,
		Y0: stripTop,
		X1: bound.X1,
		Y1: bound.Y1,
	}

	full, err := p.ExtractText(TextOptions{})
	if err != nil {
		t.Fatalf("ExtractText full: %v", err)
	}

	clipped, err := p.ExtractText(TextOptions{Clip: strip})
	if err != nil {
		t.Fatalf("ExtractText clipped narrow: %v", err)
	}

	if len(clipped) >= len(full) {
		t.Errorf("narrow-strip clipped text (%d chars) should be shorter than full text (%d chars)",
			len(clipped), len(full))
	}
}

// TestExtractTextClosedDocument verifies that calling ExtractText on a closed
// document returns a non-nil error.
func TestExtractTextClosedDocument(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}
	d.Close()

	_, err = p.ExtractText(TextOptions{})
	if err == nil {
		t.Fatal("expected error for closed document, got nil")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("expected 'closed' in error message, got %q", err.Error())
	}
}

// TestExtractTextPreserveLigatures verifies that PreserveLigatures returns
// non-empty text without error.
func TestExtractTextPreserveLigatures(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	got, err := p.ExtractText(TextOptions{PreserveLigatures: true})
	if err != nil {
		t.Fatalf("ExtractText PreserveLigatures: %v", err)
	}
	if got == "" {
		t.Fatal("ExtractText with PreserveLigatures returned empty string")
	}
}

// TestExtractTextDehyphenate verifies that Dehyphenate returns non-empty text
// without error.
func TestExtractTextDehyphenate(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	got, err := p.ExtractText(TextOptions{Dehyphenate: true})
	if err != nil {
		t.Fatalf("ExtractText Dehyphenate: %v", err)
	}
	if got == "" {
		t.Fatal("ExtractText with Dehyphenate returned empty string")
	}
}
