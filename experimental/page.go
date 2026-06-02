package experimental

import (
	"sort"
	"strings"

	"github.com/srijanmukherjee/gomupdf"
)

// Page is a positioned-text view of one PDF page. It loads the page's words
// once (lazily) and serves every spatial query from that cache.
type Page struct {
	raw *gomupdf.Page
	idx int

	loaded  bool
	loadErr error
	words   Words
}

// Word is a single positioned word. Its geometry is a unified Rect (top-left
// origin, y down, PDF points). The accessors below name the edges the way most
// layout-aware code refers to them.
type Word struct {
	Text  string
	Rect  Rect
	Block int // source block index (reading-order hint from MuPDF)
	Line  int // source line index within the block
}

// Top is the word's upper edge (smaller y).
func (w Word) Top() float64 { return w.Rect.Y0 }

// Bottom is the word's lower edge (larger y).
func (w Word) Bottom() float64 { return w.Rect.Y1 }

// Left is the word's left edge.
func (w Word) Left() float64 { return w.Rect.X0 }

// Right is the word's right edge.
func (w Word) Right() float64 { return w.Rect.X1 }

// Height is the word box height (bottom - top), useful for filtering oversized
// outliers such as watermarks.
func (w Word) Height() float64 { return w.Rect.Height() }

// Words is a slice of Word with chainable spatial filters. Methods return new
// slices and never mutate the receiver, so they compose:
//
//	row.Words.Band(lo, hi).Text()
type Words []Word

// Band keeps words whose left edge (x0) falls within [lo, hi]. This is the
// x-band filter for reading a single column.
func (ws Words) Band(lo, hi float64) Words {
	out := make(Words, 0, len(ws))
	for _, w := range ws {
		if w.Rect.X0 >= lo && w.Rect.X0 <= hi {
			out = append(out, w)
		}
	}
	return out
}

// In keeps words whose center point lies inside r.
func (ws Words) In(r Rect) Words {
	out := make(Words, 0, len(ws))
	for _, w := range ws {
		if r.ContainsPoint(w.Rect.CenterX(), w.Rect.CenterY()) {
			out = append(out, w)
		}
	}
	return out
}

// LeftOf keeps words whose left edge is strictly left of x.
func (ws Words) LeftOf(x float64) Words {
	out := make(Words, 0, len(ws))
	for _, w := range ws {
		if w.Rect.X0 < x {
			out = append(out, w)
		}
	}
	return out
}

// RightOf keeps words whose left edge is at or right of x.
func (ws Words) RightOf(x float64) Words {
	out := make(Words, 0, len(ws))
	for _, w := range ws {
		if w.Rect.X0 >= x {
			out = append(out, w)
		}
	}
	return out
}

// SortByX returns the words sorted left-to-right.
func (ws Words) SortByX() Words {
	out := append(Words(nil), ws...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rect.X0 < out[j].Rect.X0 })
	return out
}

// SortReading returns the words in reading order (top-to-bottom, then
// left-to-right), tolerating small vertical jitter within a line.
func (ws Words) SortReading() Words {
	out := append(Words(nil), ws...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if abs(a.Rect.Y0-b.Rect.Y0) > defaultRowTolerance {
			return a.Rect.Y0 < b.Rect.Y0
		}
		return a.Rect.X0 < b.Rect.X0
	})
	return out
}

// Text joins the words left-to-right with single spaces.
func (ws Words) Text() string {
	sorted := ws.SortByX()
	parts := make([]string, len(sorted))
	for i, w := range sorted {
		parts[i] = w.Text
	}
	return strings.Join(parts, " ")
}

// Bounds returns the union rect of all words (zero Rect if empty).
func (ws Words) Bounds() Rect {
	var r Rect
	for _, w := range ws {
		r = r.Union(w.Rect)
	}
	return r
}

// DropOutliers removes words whose height exceeds factor × the median word
// height — the standard trick for stripping oversized diagonal watermarks.
// A factor <= 0 defaults to 2.2.
func (ws Words) DropOutliers(factor float64) Words {
	if len(ws) == 0 {
		return ws
	}
	if factor <= 0 {
		factor = 2.2
	}
	heights := make([]float64, len(ws))
	for i, w := range ws {
		heights[i] = w.Height()
	}
	sort.Float64s(heights)
	median := heights[len(heights)/2]
	limit := median * factor
	out := make(Words, 0, len(ws))
	for _, w := range ws {
		if w.Height() <= limit {
			out = append(out, w)
		}
	}
	return out
}

// Row is a visual line of words sharing roughly the same vertical position,
// already sorted left-to-right.
type Row struct {
	Top   float64
	Words Words
}

// Text joins the row's words left-to-right with single spaces.
func (r Row) Text() string { return r.Words.Text() }

// Band returns the row's words whose left edge falls within [lo, hi].
func (r Row) Band(lo, hi float64) Words { return r.Words.Band(lo, hi) }

// Bounds returns the row's bounding rect.
func (r Row) Bounds() Rect { return r.Words.Bounds() }

// RowOption configures Rows.
type RowOption func(*rowConfig)

type rowConfig struct {
	tolerance float64
}

const defaultRowTolerance = 3.5 // points; matches typical document line spacing

// RowTolerance sets the max vertical gap (points) between a word and the
// current line before a new line starts. Default 3.5.
func RowTolerance(pts float64) RowOption {
	return func(c *rowConfig) { c.tolerance = pts }
}

func (p *Page) ensure() error {
	if p.loaded {
		return p.loadErr
	}
	p.loaded = true
	raw, err := p.raw.Words()
	if err != nil {
		p.loadErr = err
		return err
	}
	ws := make(Words, len(raw))
	for i, w := range raw {
		ws[i] = Word{Text: w.Text, Rect: rectFromMu(w.BBox), Block: w.Block, Line: w.Line}
	}
	p.words = ws
	return nil
}

// Words returns every positioned word on the page.
func (p *Page) Words() (Words, error) {
	if err := p.ensure(); err != nil {
		return nil, err
	}
	return p.words, nil
}

// Rows groups the page's words into visual lines by vertical position: words
// are swept top-to-bottom and a new line starts when a word's top exceeds the
// current line's first word by more than the tolerance. Each row's words are
// sorted left-to-right. This is the workhorse for table-style and multi-column
// layouts where native line breaks are unreliable across columns.
func (p *Page) Rows(opts ...RowOption) ([]Row, error) {
	if err := p.ensure(); err != nil {
		return nil, err
	}
	cfg := rowConfig{tolerance: defaultRowTolerance}
	for _, o := range opts {
		o(&cfg)
	}
	return clusterRows(p.words, cfg.tolerance), nil
}

// ClusterRows groups an arbitrary set of words into visual lines by vertical
// position, identically to Rows. Use it when you need to cluster a filtered or
// combined word set (e.g. after DropOutliers, or merging words from several
// sources) rather than a whole page.
func ClusterRows(ws Words, tolerance float64) []Row {
	if tolerance <= 0 {
		tolerance = defaultRowTolerance
	}
	return clusterRows(ws, tolerance)
}

// clusterRows performs the top-sweep line clustering shared by Rows and TextIn.
func clusterRows(ws Words, tol float64) []Row {
	if len(ws) == 0 {
		return nil
	}
	bySweep := append(Words(nil), ws...)
	sort.SliceStable(bySweep, func(i, j int) bool { return bySweep[i].Rect.Y0 < bySweep[j].Rect.Y0 })

	var rows []Row
	var cur Words
	var anchor float64
	for i, w := range bySweep {
		if i > 0 && w.Rect.Y0-anchor <= tol {
			cur = append(cur, w)
			continue
		}
		if len(cur) > 0 {
			rows = append(rows, Row{Top: cur[0].Rect.Y0, Words: cur.SortByX()})
		}
		cur = Words{w}
		anchor = w.Rect.Y0
	}
	if len(cur) > 0 {
		rows = append(rows, Row{Top: cur[0].Rect.Y0, Words: cur.SortByX()})
	}
	return rows
}

// WordsIn returns the words whose center lies inside the region r.
func (p *Page) WordsIn(r Rect) (Words, error) {
	if err := p.ensure(); err != nil {
		return nil, err
	}
	return p.words.In(r), nil
}

// TextIn returns the text inside region r, clustered into reading-order lines
// (rows joined by newline, words within a row by space). This is region/area
// cropping: hand it a header box, a column, a cell — get just that text.
func (p *Page) TextIn(r Rect) (string, error) {
	ws, err := p.WordsIn(r)
	if err != nil {
		return "", err
	}
	rows := clusterRows(ws, defaultRowTolerance)
	lines := make([]string, len(rows))
	for i, row := range rows {
		lines[i] = row.Text()
	}
	return strings.Join(lines, "\n"), nil
}

// Text returns the page's reading-order plain text.
func (p *Page) Text() (string, error) { return p.raw.GetText() }

// Lines returns the page's non-empty lines (soft hyphens stripped) as produced
// by gomupdf's reading-order extractor.
func (p *Page) Lines() ([]string, error) { return p.raw.Lines() }

// Bound returns the page's bounding rect in points.
func (p *Page) Bound() (Rect, error) {
	b, err := p.raw.Bound()
	if err != nil {
		return Rect{}, err
	}
	return rectFromGeometry(b), nil
}

// Raw exposes the underlying gomupdf.Page for unwrapped features (pixmap,
// images, drawings, tables, search quads, ...).
func (p *Page) Raw() *gomupdf.Page { return p.raw }

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
