package experimental

import "github.com/srijanmukherjee/gomupdf"

// Table is a detected table tagged with its source page. It embeds
// gomupdf.Table, so Rows, ColX, RowY, NumRows() and NumCols() are available
// directly; Region() gives the bounding box in the unified Rect type.
type Table struct {
	Page int
	gomupdf.Table
}

// Region returns the table's bounding box as a unified Rect.
func (t Table) Region() Rect { return rectFromMu(t.BBox) }

// TableOption configures table detection.
type TableOption func(*tableConfig)

type tableConfig struct {
	strategy *gomupdf.TableStrategy // nil = auto (text, then lines)
}

// TableText forces the word-alignment ("text") strategy.
func TableText() TableOption {
	return func(c *tableConfig) { s := gomupdf.StrategyText; c.strategy = &s }
}

// TableLines forces the vector-drawing ("lines") strategy.
func TableLines() TableOption {
	return func(c *tableConfig) { s := gomupdf.StrategyLines; c.strategy = &s }
}

// Tables detects tables on the page. By default it tries the text strategy and
// falls back to the lines strategy when text finds nothing — so callers need
// not know the document's layout up front. Force a strategy with TableText /
// TableLines.
func (p *Page) Tables(opts ...TableOption) ([]Table, error) {
	var cfg tableConfig
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.strategy != nil {
		return p.findTables(*cfg.strategy)
	}
	tabs, err := p.findTables(gomupdf.StrategyText)
	if err != nil {
		return nil, err
	}
	if len(tabs) == 0 {
		return p.findTables(gomupdf.StrategyLines)
	}
	return tabs, nil
}

func (p *Page) findTables(strategy gomupdf.TableStrategy) ([]Table, error) {
	settings := gomupdf.DefaultTableSettings()
	settings.Strategy = strategy
	raw, err := p.raw.FindTables(settings)
	if err != nil {
		return nil, err
	}
	out := make([]Table, len(raw))
	for i, t := range raw {
		out[i] = Table{Page: p.idx, Table: t}
	}
	return out, nil
}

// TablesIn returns the page's tables whose bounding box overlaps region r.
func (p *Page) TablesIn(r Rect, opts ...TableOption) ([]Table, error) {
	all, err := p.Tables(opts...)
	if err != nil {
		return nil, err
	}
	var out []Table
	for _, t := range all {
		if t.Region().Overlaps(r) {
			out = append(out, t)
		}
	}
	return out, nil
}

// Tables detects tables across every page, tagged by page.
func (d *Doc) Tables(opts ...TableOption) ([]Table, error) {
	var out []Table
	for _, page := range d.Pages() {
		tabs, err := page.Tables(opts...)
		if err != nil {
			return nil, err
		}
		out = append(out, tabs...)
	}
	return out, nil
}
