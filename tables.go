package gomupdf

import (
	"sort"
	"strings"
)

// Table detection. The "text" strategy derives column/row rulings from the
// alignment of words (no vector graphics needed), then builds a cell grid and
// extracts per-cell text.
//
// The "lines" strategy (rulings from vector drawings) needs a path-capturing
// device binding and is a documented follow-up; the text strategy already
// handles well-aligned tables.

// TableStrategy selects how column/row rulings are found.
type TableStrategy string

const (
	StrategyText  TableStrategy = "text"  // rulings from word alignment
	StrategyLines TableStrategy = "lines" // rulings from vector drawings
)

// TableSettings configures table detection; see DefaultTableSettings for the
// documented defaults.
type TableSettings struct {
	Strategy           TableStrategy // text | lines (default text)
	SnapTolerance      float64       // cluster edge coords within this distance
	MinWordsVertical   int           // words sharing an x to form a column ruling
	MinWordsHorizontal int           // words sharing a y to form a row ruling
	IntersectionTol    float64       // edge crossing tolerance
	AlignTolerance     float64       // max deviation for a drawing segment to count as axis-aligned (lines)
}

// DefaultTableSettings returns sensible defaults for table detection.
func DefaultTableSettings() TableSettings {
	return TableSettings{
		Strategy:           StrategyText,
		SnapTolerance:      3,
		MinWordsVertical:   3,
		MinWordsHorizontal: 1,
		IntersectionTol:    3,
		AlignTolerance:     2,
	}
}

// Table is a detected table: a row-major grid of cell strings plus geometry.
type Table struct {
	BBox Rect
	Rows [][]string
	ColX []float64 // column boundary x-positions (len = cols+1)
	RowY []float64 // row boundary y-positions (len = rows+1)
}

// NumRows / NumCols.
func (t *Table) NumRows() int { return len(t.Rows) }
func (t *Table) NumCols() int {
	if len(t.Rows) == 0 {
		return 0
	}
	return len(t.Rows[0])
}

type word struct {
	x0, y0, x1, y1 float64
	text           string
}

func (w word) cx() float64 { return (w.x0 + w.x1) / 2 }
func (w word) cy() float64 { return (w.y0 + w.y1) / 2 }

// FindTables detects tables on the page. Default strategy is "text" (rulings
// from word alignment); set Strategy: StrategyLines to derive rulings from the
// page's vector drawings instead.
func (p *Page) FindTables(opts ...TableSettings) ([]Table, error) {
	st := DefaultTableSettings()
	if len(opts) > 0 {
		st = opts[0]
		if st.Strategy == "" {
			st.Strategy = StrategyText
		}
	}
	// Words are always needed to fill cell text.
	pwords, err := p.Words()
	if err != nil {
		return nil, err
	}
	words := make([]word, 0, len(pwords))
	for _, w := range pwords {
		words = append(words, word{
			x0: w.BBox.X, y0: w.BBox.Y, x1: w.BBox.X1(), y1: w.BBox.Y1(), text: w.Text,
		})
	}

	var vEdges []vedge
	var hEdges []hedge
	switch st.Strategy {
	case StrategyLines:
		drawings, err := p.GetDrawings()
		if err != nil {
			return nil, err
		}
		vEdges, hEdges = drawingsToEdges(drawings, st.AlignTolerance)
	default:
		if len(words) == 0 {
			return nil, nil
		}
		vEdges = wordsToEdgesV(words, st.MinWordsVertical)
		hEdges = wordsToEdgesH(words, st.MinWordsHorizontal)
	}
	return coreTables(vEdges, hEdges, words, st), nil
}

// drawingsToEdges turns axis-aligned drawing segments (line items and rect
// sides) into vertical/horizontal rulings for the lines strategy.
func drawingsToEdges(drawings []Drawing, alignTol float64) ([]vedge, []hedge) {
	var vs []vedge
	var hs []hedge
	for _, d := range drawings {
		for _, it := range d.Items {
			if it.Op != "l" || len(it.Pts) != 2 {
				continue
			}
			a, b := it.Pts[0], it.Pts[1]
			dx, dy := abs(a.X-b.X), abs(a.Y-b.Y)
			switch {
			case dy <= alignTol && dx > alignTol: // horizontal
				hs = append(hs, hedge{y: (a.Y + b.Y) / 2, x0: minf(a.X, b.X), x1: maxf(a.X, b.X)})
			case dx <= alignTol && dy > alignTol: // vertical
				vs = append(vs, vedge{x: (a.X + b.X) / 2, y0: minf(a.Y, b.Y), y1: maxf(a.Y, b.Y)})
			}
		}
	}
	return vs, hs
}

// coreTables runs the shared edges → lattice → cells → tables pipeline.
func coreTables(vEdges []vedge, hEdges []hedge, words []word, st TableSettings) []Table {
	xs := snap(edgeXs(vEdges), st.SnapTolerance)
	ys := snap(edgeYs(hEdges), st.SnapTolerance)
	if len(xs) < 2 || len(ys) < 2 {
		return nil
	}

	// edges → intersection lattice: a grid point (xs[i], ys[j]) is real only
	// where a vertical ruling actually spans that y AND a horizontal ruling
	// spans that x (within tolerance). This is what keeps incidental alignments
	// from forming phantom cells.
	tol := st.IntersectionTol
	pt := make([][]bool, len(xs))
	for i := range pt {
		pt[i] = make([]bool, len(ys))
		for j := range pt[i] {
			pt[i][j] = vCovers(vEdges, xs[i], ys[j], tol) && hCovers(hEdges, ys[j], xs[i], tol)
		}
	}

	// intersections → cells: a cell (i,j) exists when its four corners are real.
	type cellIdx struct{ i, j int }
	cells := map[cellIdx]bool{}
	for i := 0; i+1 < len(xs); i++ {
		for j := 0; j+1 < len(ys); j++ {
			if pt[i][j] && pt[i+1][j] && pt[i][j+1] && pt[i+1][j+1] {
				cells[cellIdx{i, j}] = true
			}
		}
	}
	if len(cells) == 0 {
		return nil
	}

	// cells → tables: union-find over edge-adjacent cells (contiguous regions).
	uf := newUnionFind()
	for c := range cells {
		uf.add(c.i*100000 + c.j)
	}
	key := func(i, j int) int { return i*100000 + j }
	for c := range cells {
		if cells[cellIdx{c.i + 1, c.j}] {
			uf.union(key(c.i, c.j), key(c.i+1, c.j))
		}
		if cells[cellIdx{c.i, c.j + 1}] {
			uf.union(key(c.i, c.j), key(c.i, c.j+1))
		}
	}
	groups := map[int][]cellIdx{}
	for c := range cells {
		r := uf.find(key(c.i, c.j))
		groups[r] = append(groups[r], c)
	}

	// Explicit rulings (lines) are trustworthy, so a 1-column ruled table is
	// real; the text strategy needs ≥2 columns to suppress incidental alignments.
	minCols := 2
	if st.Strategy == StrategyLines {
		minCols = 1
	}

	var tables []Table
	for _, g := range groups {
		// bounding index range of this contiguous cell region
		minI, maxI, minJ, maxJ := g[0].i, g[0].i+1, g[0].j, g[0].j+1
		for _, c := range g {
			minI = imin(minI, c.i)
			maxI = imax(maxI, c.i+1)
			minJ = imin(minJ, c.j)
			maxJ = imax(maxJ, c.j+1)
		}
		colXs := xs[minI : maxI+1]
		rowYs := ys[minJ : maxJ+1]
		t := buildGrid(colXs, rowYs, words)
		if t.NumCols() < minCols || allBlank(t) {
			continue
		}
		tables = append(tables, t)
	}
	return tables
}

func vCovers(es []vedge, x, y, tol float64) bool {
	for _, e := range es {
		if abs(e.x-x) <= tol && y >= e.y0-tol && y <= e.y1+tol {
			return true
		}
	}
	return false
}
func hCovers(es []hedge, y, x, tol float64) bool {
	for _, e := range es {
		if abs(e.y-y) <= tol && x >= e.x0-tol && x <= e.x1+tol {
			return true
		}
	}
	return false
}

func allBlank(t Table) bool {
	for _, row := range t.Rows {
		for _, c := range row {
			if strings.TrimSpace(c) != "" {
				return false
			}
		}
	}
	return true
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// tiny union-find for cell grouping.
type unionFind struct{ parent map[int]int }

func newUnionFind() *unionFind { return &unionFind{parent: map[int]int{}} }
func (u *unionFind) add(x int) { u.parent[x] = x }
func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}
func (u *unionFind) union(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[ra] = rb
	}
}

// spansToWords splits each positioned span into whitespace-delimited words,
// apportioning the span's bbox width across characters (we only have span-level
// geometry, not per-char). Good enough for alignment clustering.
func spansToWords(spans []Span) []word {
	var out []word
	for _, s := range spans {
		t := strings.TrimRight(s.Text, " ")
		if strings.TrimSpace(t) == "" {
			continue
		}
		n := len([]rune(s.Text))
		if n == 0 {
			continue
		}
		perChar := s.BBox.W / float64(n)
		runes := []rune(s.Text)
		i := 0
		for i < len(runes) {
			// skip spaces
			for i < len(runes) && runes[i] == ' ' {
				i++
			}
			start := i
			for i < len(runes) && runes[i] != ' ' {
				i++
			}
			if i > start {
				wtext := string(runes[start:i])
				out = append(out, word{
					x0:   s.BBox.X + float64(start)*perChar,
					x1:   s.BBox.X + float64(i)*perChar,
					y0:   s.BBox.Y,
					y1:   s.BBox.Y1(),
					text: wtext,
				})
			}
		}
	}
	return out
}

// clusterList groups sorted values where consecutive gaps are <= tol.
func clusterList(xs []float64, tol float64) [][]float64 {
	if len(xs) == 0 {
		return nil
	}
	sort.Float64s(xs)
	clusters := [][]float64{{xs[0]}}
	last := xs[0]
	for _, x := range xs[1:] {
		if x > last+tol {
			clusters = append(clusters, []float64{x})
		} else {
			clusters[len(clusters)-1] = append(clusters[len(clusters)-1], x)
		}
		last = x
	}
	return clusters
}

func mean(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// clusterObjects clusters words by a coordinate key (tol), returning groups.
func clusterObjects(words []word, key func(word) float64, tol float64) [][]word {
	type kv struct {
		k float64
		w word
	}
	items := make([]kv, len(words))
	for i, w := range words {
		items[i] = kv{key(w), w}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].k < items[j].k })
	var groups [][]word
	if len(items) == 0 {
		return groups
	}
	cur := []word{items[0].w}
	last := items[0].k
	for _, it := range items[1:] {
		if it.k > last+tol {
			groups = append(groups, cur)
			cur = []word{it.w}
		} else {
			cur = append(cur, it.w)
		}
		last = it.k
	}
	groups = append(groups, cur)
	return groups
}

type vedge struct{ x, y0, y1 float64 }
type hedge struct{ y, x0, x1 float64 }

// wordsToEdgesV finds vertical rulings where >= threshold words share a left,
// right, or center x (clustered at tol 1).
func wordsToEdgesV(words []word, threshold int) []vedge {
	var candidates [][]word
	candidates = append(candidates, clusterObjects(words, func(w word) float64 { return w.x0 }, 1)...)
	candidates = append(candidates, clusterObjects(words, func(w word) float64 { return w.x1 }, 1)...)
	candidates = append(candidates, clusterObjects(words, func(w word) float64 { return w.cx() }, 1)...)

	var edges []vedge
	for _, c := range candidates {
		if len(c) < threshold {
			continue
		}
		x := c[0].x0
		y0, y1 := c[0].y0, c[0].y1
		for _, w := range c {
			if w.y0 < y0 {
				y0 = w.y0
			}
			if w.y1 > y1 {
				y1 = w.y1
			}
		}
		edges = append(edges, vedge{x: x, y0: y0, y1: y1})
	}
	return edges
}

// wordsToEdgesH finds horizontal rulings connecting the tops/bottoms of >=
// threshold words clustered by top (tol 1).
func wordsToEdgesH(words []word, threshold int) []hedge {
	groups := clusterObjects(words, func(w word) float64 { return w.y0 }, 1)
	var edges []hedge
	for _, c := range groups {
		if len(c) < threshold {
			continue
		}
		x0, x1 := c[0].x0, c[0].x1
		yTop, yBot := c[0].y0, c[0].y1
		for _, w := range c {
			if w.x0 < x0 {
				x0 = w.x0
			}
			if w.x1 > x1 {
				x1 = w.x1
			}
			if w.y0 < yTop {
				yTop = w.y0
			}
			if w.y1 > yBot {
				yBot = w.y1
			}
		}
		edges = append(edges, hedge{y: yTop, x0: x0, x1: x1})
		edges = append(edges, hedge{y: yBot, x0: x0, x1: x1})
	}
	return edges
}

func edgeXs(es []vedge) []float64 {
	out := make([]float64, len(es))
	for i, e := range es {
		out[i] = e.x
	}
	return out
}
func edgeYs(es []hedge) []float64 {
	out := make([]float64, len(es))
	for i, e := range es {
		out[i] = e.y
	}
	return out
}

// snap clusters coordinates within tol and returns each cluster's mean, sorted.
func snap(xs []float64, tol float64) []float64 {
	var out []float64
	for _, c := range clusterList(xs, tol) {
		out = append(out, mean(c))
	}
	return out
}

// buildGrid lays words into the cell grid defined by snapped x/y rulings. Each
// word is assigned to the column whose [xs[i],xs[i+1]) contains its center and
// the row whose [ys[j],ys[j+1]) contains its center; cell text joins assigned
// words left-to-right.
func buildGrid(xs, ys []float64, words []word) Table {
	cols := len(xs) - 1
	rows := len(ys) - 1
	type cell struct{ ws []word }
	grid := make([][]cell, rows)
	for r := range grid {
		grid[r] = make([]cell, cols)
	}
	colOf := func(cx float64) int {
		for i := 0; i < cols; i++ {
			if cx >= xs[i] && cx < xs[i+1] {
				return i
			}
		}
		return -1
	}
	rowOf := func(cy float64) int {
		for j := 0; j < rows; j++ {
			if cy >= ys[j] && cy < ys[j+1] {
				return j
			}
		}
		return -1
	}
	for _, w := range words {
		c := colOf(w.cx())
		r := rowOf(w.cy())
		if c >= 0 && r >= 0 {
			grid[r][c].ws = append(grid[r][c].ws, w)
		}
	}
	out := Table{
		BBox: Rect{X: xs[0], Y: ys[0], W: xs[len(xs)-1] - xs[0], H: ys[len(ys)-1] - ys[0]},
		ColX: xs,
		RowY: ys,
		Rows: make([][]string, rows),
	}
	for r := 0; r < rows; r++ {
		out.Rows[r] = make([]string, cols)
		for c := 0; c < cols; c++ {
			ws := grid[r][c].ws
			sort.Slice(ws, func(i, j int) bool { return ws[i].x0 < ws[j].x0 })
			parts := make([]string, len(ws))
			for i, w := range ws {
				parts[i] = w.text
			}
			out.Rows[r][c] = strings.Join(parts, " ")
		}
	}
	return out
}
