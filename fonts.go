package gomupdf

import (
	"errors"
	"strconv"
	"strings"
)

// FontInfo describes a font referenced by a page.
type FontInfo struct {
	Xref     int    // PDF object number of the font dictionary (use with ExtractFont)
	Name     string // BaseFont name, e.g. "Helvetica" or "ABCDEE+Calibri"
	Type     string // font Subtype, e.g. "Type1", "TrueType", "Type0"
	Encoding string // /Encoding name if present, else ""
	Embedded bool   // whether a font program is embedded (FontFile/FontFile2/FontFile3)
}

// GetFonts returns the fonts referenced by the page's /Resources /Font dict.
// The slice is deduplicated by xref (fonts with xref==0 are never deduped).
func (p *Page) GetFonts() ([]FontInfo, error) {
	d := p.doc
	d.mu.Lock()
	if d.b == nil {
		d.mu.Unlock()
		return nil, errors.New("gomupdf: document closed")
	}
	raw, err := d.b.fontsRaw(p.Number)
	d.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}

	seen := map[int]bool{}
	var fonts []FontInfo
	for _, line := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if line == "" {
			continue
		}
		// format: xref TAB embedded TAB Subtype TAB Encoding TAB BaseFont
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 5 {
			continue
		}
		xref, _ := strconv.Atoi(parts[0])
		embedded := parts[1] == "1"
		if xref != 0 && seen[xref] {
			continue
		}
		if xref != 0 {
			seen[xref] = true
		}
		fonts = append(fonts, FontInfo{
			Xref:     xref,
			Embedded: embedded,
			Type:     parts[2],
			Encoding: parts[3],
			Name:     parts[4],
		})
	}
	return fonts, nil
}

// ExtractFont returns an embedded font program by font xref: the BaseFont name,
// a file extension hint ("ttf", "cff", "otf", "pfa"), and the raw bytes.
// Returns empty bytes (no error) when the font is not embedded.
func (d *Document) ExtractFont(xref int) (name, ext string, data []byte, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", "", nil, errors.New("gomupdf: document closed")
	}
	return d.b.extractFont(xref)
}
