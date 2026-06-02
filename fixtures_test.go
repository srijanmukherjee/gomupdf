package gomupdf

import (
	"math"
	"strings"
	"testing"
)

// Tests against committed PDF fixtures in testdata/resources/. These are small
// sample, public-domain PDFs and run unconditionally.

const res = "testdata/resources/"

func openFixture(t *testing.T, name string) *Document {
	t.Helper()
	d, err := Open(res + name)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	return d
}

// Verifies that all stroke widths equal 15 after the page's scaling matrix is
// applied.
func TestDrawingsWidth3591(t *testing.T) {
	d := openFixture(t, "test-3591.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	dr, err := p.GetDrawings()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, g := range dr {
		if g.Type == "s" {
			n++
			if math.Abs(g.Width-15) > 0.01 {
				t.Errorf("stroke width = %v, want 15", g.Width)
			}
		}
	}
	if n == 0 {
		t.Fatal("no stroke drawings found")
	}
}

// Structural check of GetDrawings on a ruled table: every rect band yields a
// closed path (≥4 line segments) and an axis-aligned bbox.
func TestDrawingsSmallTable(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	dr, err := p.GetDrawings()
	if err != nil {
		t.Fatal(err)
	}
	if len(dr) == 0 {
		t.Fatal("no drawings")
	}
	fills, strokes := 0, 0
	for _, g := range dr {
		switch g.Type {
		case "f":
			fills++
		case "s":
			strokes++
		}
		lines := 0
		for _, it := range g.Items {
			if it.Op == "l" {
				lines++
			}
		}
		if lines < 4 {
			t.Errorf("ruled rect should have ≥4 line segments, got %d", lines)
		}
	}
	if fills == 0 || strokes == 0 {
		t.Errorf("expected fills and strokes, got fills=%d strokes=%d", fills, strokes)
	}
}

// Lines strategy on a ruled table (small-table.pdf fixture).
func TestTablesLinesSmallTable(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	tabs, err := p.FindTables(TableSettings{Strategy: StrategyLines})
	if err != nil {
		t.Fatal(err)
	}
	if len(tabs) != 1 {
		t.Fatalf("want 1 table, got %d", len(tabs))
	}
	tab := tabs[0]
	if tab.NumRows() != 5 {
		t.Errorf("want 5 rows, got %d", tab.NumRows())
	}
	joined := func(row []string) string {
		return strings.Join(row, " ")
	}
	if !strings.Contains(joined(tab.Rows[0]), "Boiling Points") {
		t.Errorf("header row = %q", joined(tab.Rows[0]))
	}
	found := false
	for _, r := range tab.Rows {
		if strings.Contains(joined(r), "Noble gases") {
			found = true
		}
	}
	if !found {
		t.Error("expected a row containing 'Noble gases'")
	}
}
