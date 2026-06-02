package gomupdf

import (
	"strconv"
	"strings"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// Search finds all occurrences of needle on the page and returns their hit
// quads. Matching is MuPDF's case-insensitive, whitespace-tolerant search.
// Returns nil if there are no hits.
func (p *Page) Search(needle string) ([]geometry.Quad, error) {
	if needle == "" {
		return nil, nil
	}
	raw, err := p.searchRaw(needle)
	if err != nil {
		return nil, err
	}
	var out []geometry.Quad
	for _, ln := range strings.Split(raw, "\n") {
		f := strings.Fields(ln)
		if len(f) != 8 {
			continue
		}
		v := make([]float64, 8)
		ok := true
		for i := 0; i < 8; i++ {
			if v[i], err = strconv.ParseFloat(f[i], 64); err != nil {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		out = append(out, geometry.Quad{
			UL: geometry.Point{X: v[0], Y: v[1]},
			UR: geometry.Point{X: v[2], Y: v[3]},
			LL: geometry.Point{X: v[4], Y: v[5]},
			LR: geometry.Point{X: v[6], Y: v[7]},
		})
	}
	return out, nil
}

// SearchRects is Search returning bounding rectangles instead of quads.
func (p *Page) SearchRects(needle string) ([]geometry.Rect, error) {
	quads, err := p.Search(needle)
	if err != nil {
		return nil, err
	}
	out := make([]geometry.Rect, len(quads))
	for i, q := range quads {
		out[i] = q.Rect()
	}
	return out, nil
}
