package gomupdf

import (
	"errors"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// TextOptions controls how text is extracted from a page.
type TextOptions struct {
	Clip               *geometry.Rect // if non-nil, only text intersecting this rect (PDF points) is returned
	Dehyphenate        bool           // join words split by a soft hyphen at line end
	PreserveWhitespace bool           // keep original whitespace instead of normalizing
	PreserveLigatures  bool           // keep ligatures (fi, fl, …) instead of expanding
	PreserveImages     bool           // keep image placeholders in the output
	InhibitSpaces      bool           // do not synthesize spaces between glyphs
}

// ExtractText returns the page's text honoring opts (reading order, lines separated by '\n').
func (p *Page) ExtractText(opts TextOptions) (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	return d.b.extractTextRaw(p.Number, opts)
}
