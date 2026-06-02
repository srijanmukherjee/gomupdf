package gomupdf

import (
	"strings"
	"testing"
)

// makeTable constructs a Table directly from a slice of rows (no PDF needed).
func makeTable(rows [][]string) Table {
	if len(rows) == 0 {
		return Table{}
	}
	cols := len(rows[0])
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	colX := make([]float64, cols+1)
	for i := range colX {
		colX[i] = float64(i) * 100
	}
	rowY := make([]float64, len(rows)+1)
	for i := range rowY {
		rowY[i] = float64(i) * 20
	}
	return Table{
		Rows: rows,
		ColX: colX,
		RowY: rowY,
	}
}

func TestToMarkdown_3x3(t *testing.T) {
	tab := makeTable([][]string{
		{"Name", "Age", "City"},
		{"Alice", "30", "Paris"},
		{"Bob", "25", "London"},
	})
	got := tab.ToMarkdown()
	want := "| Name | Age | City |\n| --- | --- | --- |\n| Alice | 30 | Paris |\n| Bob | 25 | London |"
	if got != want {
		t.Errorf("ToMarkdown() 3x3\ngot:  %q\nwant: %q", got, want)
	}
}

func TestToMarkdown_SingleRow(t *testing.T) {
	tab := makeTable([][]string{
		{"Name", "Age", "City"},
	})
	got := tab.ToMarkdown()
	want := "| Name | Age | City |\n| --- | --- | --- |"
	if got != want {
		t.Errorf("ToMarkdown() single row\ngot:  %q\nwant: %q", got, want)
	}
}

func TestToMarkdown_Empty(t *testing.T) {
	tab := makeTable(nil)
	got := tab.ToMarkdown()
	if got != "" {
		t.Errorf("ToMarkdown() empty: got %q, want \"\"", got)
	}
}

func TestToMarkdown_EmptyRows(t *testing.T) {
	tab := Table{Rows: [][]string{}}
	got := tab.ToMarkdown()
	if got != "" {
		t.Errorf("ToMarkdown() empty rows: got %q, want \"\"", got)
	}
}

func TestToMarkdown_PipeEscaped(t *testing.T) {
	tab := makeTable([][]string{
		{"Key", "Value"},
		{"a|b", "c"},
	})
	got := tab.ToMarkdown()
	want := `| Key | Value |` + "\n" + `| --- | --- |` + "\n" + `| a\|b | c |`
	if got != want {
		t.Errorf("ToMarkdown() pipe escape\ngot:  %q\nwant: %q", got, want)
	}
}

func TestToMarkdown_RaggedRowPadded(t *testing.T) {
	// Header has 3 cols; second row only has 2 — should be padded with empty cell.
	tab := Table{
		Rows: [][]string{
			{"A", "B", "C"},
			{"1", "2"},
		},
		ColX: []float64{0, 100, 200, 300},
		RowY: []float64{0, 20, 40},
	}
	got := tab.ToMarkdown()
	want := "| A | B | C |\n| --- | --- | --- |\n| 1 | 2 |  |"
	if got != want {
		t.Errorf("ToMarkdown() ragged row\ngot:  %q\nwant: %q", got, want)
	}
}

func TestToMarkdown_NewlineInCell(t *testing.T) {
	tab := makeTable([][]string{
		{"Header"},
		{"line1\nline2"},
	})
	got := tab.ToMarkdown()
	if strings.Contains(got, "\n\n") {
		t.Errorf("ToMarkdown() should collapse cell newlines, got: %q", got)
	}
	if !strings.Contains(got, "line1 line2") {
		t.Errorf("ToMarkdown() newline in cell not collapsed to space, got: %q", got)
	}
}

func TestHeader_Normal(t *testing.T) {
	tab := makeTable([][]string{
		{"Name", "Age"},
		{"Alice", "30"},
	})
	h := tab.Header()
	if len(h) != 2 || h[0] != "Name" || h[1] != "Age" {
		t.Errorf("Header() = %v, want [Name Age]", h)
	}
}

func TestHeader_Empty(t *testing.T) {
	tab := makeTable(nil)
	h := tab.Header()
	if h != nil {
		t.Errorf("Header() empty table = %v, want nil", h)
	}
}

func TestHeader_SingleRow(t *testing.T) {
	tab := makeTable([][]string{{"X", "Y", "Z"}})
	h := tab.Header()
	if len(h) != 3 || h[0] != "X" {
		t.Errorf("Header() single row = %v", h)
	}
}

// TestToMarkdown_Integration opens a real PDF fixture and verifies that
// ToMarkdown produces valid GFM output if any tables are detected.
func TestToMarkdown_Integration(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	tabs, err := p.FindTables(TableSettings{Strategy: StrategyLines})
	if err != nil {
		t.Fatal(err)
	}
	if len(tabs) == 0 {
		// Try text strategy
		tabs, err = p.FindTables()
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(tabs) == 0 {
		t.Skip("no tables detected in fixture, skipping integration test")
	}
	md := tabs[0].ToMarkdown()
	if md == "" {
		t.Error("ToMarkdown() returned empty string for detected table")
	}
	if !strings.Contains(md, "| --- |") {
		t.Errorf("ToMarkdown() output missing separator row, got:\n%s", md)
	}
}
