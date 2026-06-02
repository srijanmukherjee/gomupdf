package experimental

import (
	"regexp"
	"strings"
)

// Match is a located text hit: the matched text and its bounding rect, plus the
// page it came from and the visual line it sat on (for context).
type Match struct {
	Page int
	Text string
	Rect Rect

	row Row // the line this match sat on
}

// Context returns the full text of the line the match sat on — a cheap snippet
// of surrounding context.
func (m Match) Context() string { return m.row.Text() }

// QueryOption configures Find / ValueRightOf / ValueBelow.
type QueryOption func(*queryConfig)

type queryConfig struct {
	caseSensitive bool
	maxGap        float64 // 0 = take the rest of the line
	pad           float64 // x-band padding for ValueBelow
}

func newQueryConfig(opts []QueryOption) queryConfig {
	c := queryConfig{pad: 2}
	for _, o := range opts {
		o(&c)
	}
	return c
}

// CaseSensitive makes label/pattern matching case-sensitive (default: insensitive).
func CaseSensitive() QueryOption {
	return func(c *queryConfig) { c.caseSensitive = true }
}

// MaxGap stops a right-of value scan when the horizontal gap between two words
// exceeds pts, so a far-away column is not swept into the value.
func MaxGap(pts float64) QueryOption {
	return func(c *queryConfig) { c.maxGap = pts }
}

// Pad widens the x-band used by ValueBelow by pts on each side (default 2).
func Pad(pts float64) QueryOption {
	return func(c *queryConfig) { c.pad = pts }
}

func compileQuery(pattern string, cs bool) (*regexp.Regexp, error) {
	if !cs {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// Find returns every regex match on the page, each with its bounding rect.
// Matching runs per visual line, so a pattern spanning adjacent words on the
// same line is found and its rect is the union of those words.
//
//	hits := page.Find(`\d{4}-\d{2}-\d{2}`) // dates, with their locations
func (p *Page) Find(pattern string, opts ...QueryOption) ([]Match, error) {
	cfg := newQueryConfig(opts)
	re, err := compileQuery(pattern, cfg.caseSensitive)
	if err != nil {
		return nil, err
	}
	rows, err := p.Rows()
	if err != nil {
		return nil, err
	}
	var out []Match
	for _, row := range rows {
		out = append(out, findInRow(row, re, p.idx)...)
	}
	return out, nil
}

// Search finds a literal substring on the page and returns each hit's location.
// Use Find for regular expressions.
func (p *Page) Search(needle string, opts ...QueryOption) ([]Match, error) {
	return p.Find(regexp.QuoteMeta(needle), opts...)
}

// findInRow runs re against a row's joined text and maps each match back to the
// union rect of the words it covers.
func findInRow(row Row, re *regexp.Regexp, page int) []Match {
	if len(row.Words) == 0 {
		return nil
	}
	type seg struct {
		start, end int
		rect       Rect
	}
	var sb strings.Builder
	segs := make([]seg, 0, len(row.Words))
	for i, w := range row.Words {
		if i > 0 {
			sb.WriteByte(' ')
		}
		start := sb.Len()
		sb.WriteString(w.Text)
		segs = append(segs, seg{start, sb.Len(), w.Rect})
	}
	text := sb.String()

	var matches []Match
	for _, loc := range re.FindAllStringIndex(text, -1) {
		s, e := loc[0], loc[1]
		var r Rect
		for _, sg := range segs {
			if sg.end > s && sg.start < e { // word overlaps the match span
				r = r.Union(sg.rect)
			}
		}
		matches = append(matches, Match{Page: page, Text: text[s:e], Rect: r, row: row})
	}
	return matches
}

// ValueRightOf finds the label on the page and returns the text immediately to
// its right on the same line — the bread-and-butter of key/value extraction:
//
//	v, ok := page.ValueRightOf("Order No")
//	v, ok := page.ValueRightOf("Total", experimental.MaxGap(40))
//
// The label may be a regular expression. ok is false if the label is not found.
func (p *Page) ValueRightOf(label string, opts ...QueryOption) (string, bool, error) {
	cfg := newQueryConfig(opts)
	re, err := compileQuery(label, cfg.caseSensitive)
	if err != nil {
		return "", false, err
	}
	rows, err := p.Rows()
	if err != nil {
		return "", false, err
	}
	for _, row := range rows {
		hits := findInRow(row, re, p.idx)
		if len(hits) == 0 {
			continue
		}
		anchor := hits[0].Rect
		right := row.Words.RightOf(anchor.X1 - 0.1).SortByX()
		// Drop words that are part of the label itself.
		filtered := right[:0:0]
		for _, w := range right {
			if w.Rect.X0 >= anchor.X1-0.1 {
				filtered = append(filtered, w)
			}
		}
		value := joinWithGap(filtered, anchor.X1, cfg.maxGap)
		value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), ":"))
		if value != "" {
			return value, true, nil
		}
	}
	return "", false, nil
}

// ValueBelow finds the label on the page and returns the text on the nearest
// line below it that sits within the label's horizontal band — for values
// printed under their heading rather than beside it.
func (p *Page) ValueBelow(label string, opts ...QueryOption) (string, bool, error) {
	cfg := newQueryConfig(opts)
	re, err := compileQuery(label, cfg.caseSensitive)
	if err != nil {
		return "", false, err
	}
	rows, err := p.Rows()
	if err != nil {
		return "", false, err
	}
	for i, row := range rows {
		hits := findInRow(row, re, p.idx)
		if len(hits) == 0 {
			continue
		}
		anchor := hits[0].Rect
		lo, hi := anchor.X0-cfg.pad, anchor.X1+cfg.pad
		for _, below := range rows[i+1:] {
			band := below.Band(lo, hi)
			if len(band) == 0 {
				continue
			}
			if v := strings.TrimSpace(band.Text()); v != "" {
				return v, true, nil
			}
		}
		return "", false, nil
	}
	return "", false, nil
}

// joinWithGap joins words left-to-right, stopping when the horizontal gap from
// the previous word's right edge exceeds maxGap (maxGap <= 0 disables the cut).
func joinWithGap(ws Words, startX, maxGap float64) string {
	prevRight := startX
	var parts []string
	for _, w := range ws {
		if maxGap > 0 && w.Rect.X0-prevRight > maxGap {
			break
		}
		parts = append(parts, w.Text)
		prevRight = w.Rect.X1
	}
	return strings.Join(parts, " ")
}

// BlockOptions configures CollectBlock.
type BlockOptions struct {
	Lo, Hi    float64        // x-band the block lives in
	MaxLines  int            // cap on collected lines (default 7)
	MaxGap    int            // empty lines tolerated mid-block (default 1)
	Stop      *regexp.Regexp // stop when a band word matches (e.g. next field label)
	GuardLeft bool           // also stop when a word left of the band matches Stop
}

// CollectBlock walks rows downward from start, collecting the text of words
// inside the x-band [Lo, Hi] until it hits a Stop keyword, runs past MaxGap
// empty lines, or reaches MaxLines. It captures the gnarly multi-line region
// reads (postal addresses, label-anchored blocks) that layout-aware extraction
// otherwise hand-rolls every time.
func CollectBlock(rows []Row, start int, opt BlockOptions) []string {
	if opt.MaxLines <= 0 {
		opt.MaxLines = 7
	}
	if opt.MaxGap <= 0 {
		opt.MaxGap = 1
	}
	var out []string
	gap := 0
	for _, row := range rows[min(start, len(rows)):] {
		band := row.Band(opt.Lo, opt.Hi)
		if opt.Stop != nil {
			if opt.GuardLeft && wordsMatch(row.Words.LeftOf(opt.Lo), opt.Stop) {
				break
			}
			if wordsMatch(band, opt.Stop) {
				break
			}
		}
		if len(band) == 0 {
			gap++
			if gap > opt.MaxGap {
				break
			}
			continue
		}
		gap = 0
		out = append(out, band.Text())
		if len(out) >= opt.MaxLines {
			break
		}
	}
	return out
}

// wordsMatch reports whether any word matches re.
func wordsMatch(ws Words, re *regexp.Regexp) bool {
	for _, w := range ws {
		if re.MatchString(w.Text) {
			return true
		}
	}
	return false
}

// Find scans every page and returns all matches, tagged by page.
func (d *Doc) Find(pattern string, opts ...QueryOption) ([]Match, error) {
	var out []Match
	for _, page := range d.Pages() {
		hits, err := page.Find(pattern, opts...)
		if err != nil {
			return nil, err
		}
		out = append(out, hits...)
	}
	return out, nil
}

// ValueRightOf scans pages in order and returns the first label/value hit.
func (d *Doc) ValueRightOf(label string, opts ...QueryOption) (string, bool, error) {
	for _, page := range d.Pages() {
		if v, ok, err := page.ValueRightOf(label, opts...); err != nil || ok {
			return v, ok, err
		}
	}
	return "", false, nil
}

// ValueBelow scans pages in order and returns the first label/value hit.
func (d *Doc) ValueBelow(label string, opts ...QueryOption) (string, bool, error) {
	for _, page := range d.Pages() {
		if v, ok, err := page.ValueBelow(label, opts...); err != nil || ok {
			return v, ok, err
		}
	}
	return "", false, nil
}
