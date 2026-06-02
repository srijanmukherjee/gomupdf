package gomupdf

import "strings"

// ToMarkdown renders the table as a GitHub-Flavored-Markdown table. The first
// row is treated as the header. Cell pipes and newlines are escaped so the
// output stays a valid single GFM table. Returns "" for an empty table.
func (t *Table) ToMarkdown() string {
	if t.NumRows() == 0 || t.NumCols() == 0 {
		return ""
	}
	cols := t.NumCols()

	renderRow := func(cells []string) string {
		parts := make([]string, cols)
		for i := 0; i < cols; i++ {
			if i < len(cells) {
				parts[i] = escapeCell(cells[i])
			}
			// empty string for padded cells — already zero value
		}
		return "| " + strings.Join(parts, " | ") + " |"
	}

	// separator row
	seps := make([]string, cols)
	for i := range seps {
		seps[i] = "---"
	}
	separator := "| " + strings.Join(seps, " | ") + " |"

	var sb strings.Builder
	sb.WriteString(renderRow(t.Rows[0]))
	sb.WriteByte('\n')
	sb.WriteString(separator)
	for _, row := range t.Rows[1:] {
		sb.WriteByte('\n')
		sb.WriteString(renderRow(row))
	}
	return sb.String()
}

// Header returns the table's header row (the first row), or nil if the table
// is empty.
func (t *Table) Header() []string {
	if t.NumRows() == 0 {
		return nil
	}
	return t.Rows[0]
}

// escapeCell escapes pipe characters and collapses newlines/carriage returns,
// then trims surrounding whitespace from a table cell value.
func escapeCell(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	return s
}
