package gomupdf

import (
	"errors"
	"strconv"
	"strings"
)

// SetTOC replaces the document outline (bookmarks) with the given flat,
// depth-first entries. Level is 1-based (1 = top level); each entry's Level
// must be <= previous level + 1. Page is 0-based; out-of-range pages clamp to
// the nearest valid page. Passing nil removes the outline. Effective on Save.
func (d *Document) SetTOC(entries []TOCEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}

	var b strings.Builder
	replacer := strings.NewReplacer("\t", " ", "\n", " ")
	for _, e := range entries {
		b.WriteString(strconv.Itoa(e.Level))
		b.WriteByte('\t')
		b.WriteString(strconv.Itoa(e.Page))
		b.WriteByte('\t')
		b.WriteString(replacer.Replace(e.Title))
		b.WriteByte('\n')
	}

	return d.b.setTOC(b.String())
}
