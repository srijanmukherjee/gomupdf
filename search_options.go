package gomupdf

import "github.com/srijanmukherjee/gomupdf/geometry"

// SearchOptions refines a page search.
type SearchOptions struct {
	// Clip, if non-nil, restricts results to quads whose bounding rectangle
	// intersects this region.
	Clip *geometry.Rect

	// MaxHits, if > 0, caps the number of returned hits.
	MaxHits int
}

// SearchWith finds all occurrences of needle on the page like Search, then
// applies opts: if opts.Clip is non-nil only hits whose bounding rect
// intersects the clip region are kept, and if opts.MaxHits > 0 at most that
// many hits are returned. Hits are in page order.
func (p *Page) SearchWith(needle string, opts SearchOptions) ([]geometry.Quad, error) {
	quads, err := p.Search(needle)
	if err != nil {
		return nil, err
	}

	if opts.Clip != nil {
		filtered := quads[:0]
		for _, q := range quads {
			if q.Rect().Intersects(*opts.Clip) {
				filtered = append(filtered, q)
			}
		}
		quads = filtered
	}

	if opts.MaxHits > 0 && len(quads) > opts.MaxHits {
		quads = quads[:opts.MaxHits]
	}

	return quads, nil
}
