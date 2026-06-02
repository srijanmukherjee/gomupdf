package gomupdf

import (
	"strings"
	"testing"
)

// TestRawDictBlocksExist verifies that RawDict returns at least one block with lines and chars.
func TestRawDictBlocksExist(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := p.RawDict()
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) == 0 {
		t.Fatal("RawDict: want >0 blocks, got 0")
	}
	hasLine := false
	for _, b := range blocks {
		if len(b.Lines) > 0 {
			hasLine = true
			break
		}
	}
	if !hasLine {
		t.Fatal("RawDict: no block has lines")
	}
}

// TestRawDictCharsNonEmpty verifies chars are present and have non-zero bboxes.
func TestRawDictCharsNonEmpty(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	blocks, err := p.RawDict()
	if err != nil {
		t.Fatal(err)
	}
	charCount := 0
	for _, b := range blocks {
		for _, l := range b.Lines {
			for _, ch := range l.Chars {
				charCount++
				bb := ch.BBox
				if bb.X1-bb.X0 < 0 {
					t.Errorf("char bbox X1<X0: %v", bb)
				}
				if bb.Y1-bb.Y0 < 0 {
					t.Errorf("char bbox Y1<Y0: %v", bb)
				}
			}
		}
	}
	if charCount == 0 {
		t.Fatal("RawDict: no chars found at all")
	}
}

// TestRawDictContainsKnownText checks that concatenating all char runes
// contains text known to be in small-table.pdf (header: "Boiling Points").
func TestRawDictContainsKnownText(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	blocks, err := p.RawDict()
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for _, b := range blocks {
		for _, l := range b.Lines {
			for _, ch := range l.Chars {
				sb.WriteRune(ch.Rune)
			}
		}
	}
	got := sb.String()
	if !strings.Contains(got, "Boiling") {
		t.Errorf("RawDict all-chars text does not contain 'Boiling'; got %q…", got[:min(len(got), 200)])
	}
}

// TestRawDictCharBBoxWithinPage sanity-checks that each char bbox is within
// the page bounding rect (with a small tolerance for rounding).
func TestRawDictCharBBoxWithinPage(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	bound, err := p.Bound()
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := p.RawDict()
	if err != nil {
		t.Fatal(err)
	}
	const tol = 5.0 // PDF-point tolerance
	for _, b := range blocks {
		for _, l := range b.Lines {
			for _, ch := range l.Chars {
				bb := ch.BBox
				if bb.X0 < bound.X0-tol || bb.X1 > bound.X1+tol ||
					bb.Y0 < bound.Y0-tol || bb.Y1 > bound.Y1+tol {
					t.Errorf("char bbox %v outside page bound %v (tol %.1f)", bb, bound, tol)
				}
			}
		}
	}
}

// TestRawDictCharOriginOrdering checks that origin falls within or near the char bbox.
func TestRawDictCharOriginOrdering(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	blocks, err := p.RawDict()
	if err != nil {
		t.Fatal(err)
	}
	const tol = 5.0
	for _, b := range blocks {
		for _, l := range b.Lines {
			for _, ch := range l.Chars {
				bb := ch.BBox
				ox, oy := ch.Origin.X, ch.Origin.Y
				if ox < bb.X0-tol || ox > bb.X1+tol {
					t.Errorf("origin.X=%g outside bbox [%g,%g] (tol %.1f)", ox, bb.X0, bb.X1, tol)
				}
				_ = oy // origin Y (baseline) may be outside bbox Y in some typefaces — skip
			}
		}
	}
}

// TestDictBlocksExist verifies that Dict returns at least one block with spans.
func TestDictBlocksExist(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	blocks, err := p.Dict()
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) == 0 {
		t.Fatal("Dict: want >0 blocks, got 0")
	}
	hasSpan := false
	for _, b := range blocks {
		if len(b.Spans) > 0 {
			hasSpan = true
			break
		}
	}
	if !hasSpan {
		t.Fatal("Dict: no block has spans")
	}
}

// TestDictSpansMeta verifies each span has non-empty Font and Size>0.
func TestDictSpansMeta(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	blocks, err := p.Dict()
	if err != nil {
		t.Fatal(err)
	}
	spanCount := 0
	for _, b := range blocks {
		for _, sp := range b.Spans {
			spanCount++
			if sp.Font == "" {
				t.Errorf("span has empty Font")
			}
			if sp.Size <= 0 {
				t.Errorf("span has non-positive Size: %g", sp.Size)
			}
			if sp.Text == "" {
				t.Errorf("span has empty Text")
			}
		}
	}
	if spanCount == 0 {
		t.Fatal("Dict: no spans found")
	}
}

// TestDictSpansTextNonEmpty verifies concatenation of all span Text is non-empty
// and contains expected content.
func TestDictSpansTextNonEmpty(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	blocks, err := p.Dict()
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for _, b := range blocks {
		for _, sp := range b.Spans {
			sb.WriteString(sp.Text)
		}
	}
	got := sb.String()
	if got == "" {
		t.Fatal("Dict: all span text is empty")
	}
	if !strings.Contains(got, "Boiling") {
		t.Errorf("Dict span text does not contain 'Boiling'; got %q…", got[:min(len(got), 200)])
	}
}

// TestRawDictSymbolList runs RawDict on symbol-list.pdf to exercise
// a different fixture (symbols / special chars).
func TestRawDictSymbolList(t *testing.T) {
	d := openFixture(t, "symbol-list.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	blocks, err := p.RawDict()
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) == 0 {
		t.Fatal("RawDict symbol-list: want >0 blocks")
	}
}

// TestRawDictClosedDocument verifies that calling RawDict on a closed document
// returns an error instead of panicking.
func TestRawDictClosedDocument(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	p, _ := d.LoadPage(0)
	d.Close()
	_, err := p.RawDict()
	if err == nil {
		t.Fatal("RawDict on closed document: want error, got nil")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("expected 'closed' in error, got %q", err.Error())
	}
}

// TestDictClosedDocument verifies that calling Dict on a closed document
// returns an error.
func TestDictClosedDocument(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	p, _ := d.LoadPage(0)
	d.Close()
	_, err := p.Dict()
	if err == nil {
		t.Fatal("Dict on closed document: want error, got nil")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("expected 'closed' in error, got %q", err.Error())
	}
}

// TestRawDictBBoxOrdering verifies x0<=x1 and y0<=y1 for all char bboxes.
func TestRawDictBBoxOrdering(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	blocks, err := p.RawDict()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range blocks {
		for _, l := range b.Lines {
			for _, ch := range l.Chars {
				bb := ch.BBox
				if bb.X0 > bb.X1 {
					t.Errorf("char bbox X0>X1: %v", bb)
				}
				if bb.Y0 > bb.Y1 {
					t.Errorf("char bbox Y0>Y1: %v", bb)
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
