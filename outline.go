package gomupdf

import (
	"strconv"
	"strings"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// TOCEntry is one table-of-contents item: a heading level, its title, and the
// destination page. Page is 0-based, or -1 for external/unresolved destinations.
type TOCEntry struct {
	Level int
	Title string
	Page  int
}

// TOC returns the document outline as a flat, depth-first list. Empty if the
// document has no bookmarks.
func (d *Document) TOC() ([]TOCEntry, error) {
	raw, err := d.tocRaw()
	if err != nil {
		return nil, err
	}
	var out []TOCEntry
	for _, ln := range strings.Split(raw, "\n") {
		if ln == "" {
			continue
		}
		parts := strings.SplitN(ln, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		level, _ := strconv.Atoi(parts[0])
		page, _ := strconv.Atoi(parts[1])
		out = append(out, TOCEntry{Level: level, Title: parts[2], Page: page})
	}
	return out, nil
}

// Link is a page link: a clickable rect with a target URI.
type Link struct {
	Rect geometry.Rect
	URI  string
}

// Links returns the page's links.
func (p *Page) Links() ([]Link, error) {
	raw, err := p.linksRaw()
	if err != nil {
		return nil, err
	}
	var out []Link
	for _, ln := range strings.Split(raw, "\n") {
		if ln == "" {
			continue
		}
		tab := strings.IndexByte(ln, '\t')
		if tab < 0 {
			continue
		}
		f := strings.Fields(ln[:tab])
		if len(f) != 4 {
			continue
		}
		x0, _ := strconv.ParseFloat(f[0], 64)
		y0, _ := strconv.ParseFloat(f[1], 64)
		x1, _ := strconv.ParseFloat(f[2], 64)
		y1, _ := strconv.ParseFloat(f[3], 64)
		out = append(out, Link{
			Rect: geometry.Rect{X0: x0, Y0: y0, X1: x1, Y1: y1},
			URI:  ln[tab+1:],
		})
	}
	return out, nil
}
