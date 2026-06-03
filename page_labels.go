package gomupdf

import (
	"errors"
	"strconv"
	"strings"
)

// Page labels: the logical (displayed) page numbering carried in the Catalog's
// /PageLabels number tree — e.g. front matter as "i, ii, iii" followed by body
// pages "1, 2, 3" or prefixed labels like "A-1". These mirror PyMuPDF's
// get_page_labels / set_page_labels / page.get_label.

// Page-label numbering styles, as used in PageLabel.Style. An empty style means
// the page has no number (prefix only).
const (
	LabelDecimal    = "D" // 1, 2, 3, …
	LabelRomanUpper = "R" // I, II, III, …
	LabelRomanLower = "r" // i, ii, iii, …
	LabelAlphaUpper = "A" // A, B, C, …, AA, …
	LabelAlphaLower = "a" // a, b, c, …, aa, …
)

// PageLabel is one numbering rule. It takes effect at StartPage (0-based) and
// applies until the next rule's StartPage. The displayed label for a page is
// Prefix followed by the page's number rendered in Style, counting from Start.
type PageLabel struct {
	StartPage int    // 0-based page index where this rule begins
	Style     string // one of the Label* constants, or "" for no number
	Prefix    string // literal text placed before the number
	Start     int    // first numeric value for the run (default 1 when < 1)
}

// SetPageLabels installs the document's page-label rules, replacing any
// existing ones. Rules must be ordered by StartPage and the first rule should
// start at page 0. Passing nil removes all page labels. Changes take effect on
// the next Save.
func (d *Document) SetPageLabels(labels []PageLabel) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	var b strings.Builder
	for _, l := range labels {
		start := l.Start
		if start < 1 {
			start = 1
		}
		// Record layout: startpage \t style \t start \t prefix.
		// Prefix is last so it may contain spaces; tabs/newlines are stripped.
		prefix := strings.NewReplacer("\t", " ", "\n", " ").Replace(l.Prefix)
		b.WriteString(strconv.Itoa(l.StartPage))
		b.WriteByte('\t')
		b.WriteString(l.Style)
		b.WriteByte('\t')
		b.WriteString(strconv.Itoa(start))
		b.WriteByte('\t')
		b.WriteString(prefix)
		b.WriteByte('\n')
	}
	return d.b.setPageLabels(b.String())
}

// PageLabels returns the document's page-label rules in page order, or nil when
// the document defines none.
func (d *Document) PageLabels() ([]PageLabel, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	raw, err := d.b.pageLabelsRaw()
	if err != nil {
		return nil, err
	}
	var out []PageLabel
	for _, ln := range strings.Split(raw, "\n") {
		if ln == "" {
			continue
		}
		// Split into the 3 leading fields plus the prefix remainder.
		parts := strings.SplitN(ln, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		sp, _ := strconv.Atoi(parts[0])
		start, _ := strconv.Atoi(parts[2])
		out = append(out, PageLabel{StartPage: sp, Style: parts[1], Start: start, Prefix: parts[3]})
	}
	return out, nil
}

// Label returns the page's resolved logical label (e.g. "ii" or "A-3"). When
// the document defines no page labels, it returns the empty string.
func (p *Page) Label() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.pageLabel(p.Number)
}
