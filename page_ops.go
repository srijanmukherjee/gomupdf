package gomupdf

import "errors"

// CopyPage inserts a shallow copy of page from (0-based) so the copy lands at
// index to (0-based). Use PageCount() as to in order to append at the end.
// The copy shares the source page's content stream (shallow reference),
// mirroring PyMuPDF's copy_page behaviour.
// Returns an error if either index is out of range or the document is closed.
func (d *Document) CopyPage(from, to int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.copyPage(from, to)
}

// MovePage moves page from (0-based) to index to (0-based).
// After the move the page that was at from will be at to (considering the
// shift caused by the removal). Returns an error if either index is out of
// range or the document is closed.
func (d *Document) MovePage(from, to int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.movePage(from, to)
}

// SelectPages rebuilds the document to contain only the given pages, in the
// given order (0-based indices; may repeat to duplicate pages). This is the
// fallback implementation — pdf_rearrange_pages is not present in this MuPDF
// build. Returns an error on invalid indices or a closed document.
func (d *Document) SelectPages(pages []int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	if len(pages) == 0 {
		return errors.New("gomupdf: SelectPages: pages list is empty")
	}
	return d.b.selectPages(pages)
}

// InsertPDFRange appends pages [fromPage, toPage] (0-based, inclusive) of the
// source PDF (provided as raw bytes) to the end of this document. password
// unlocks an encrypted source; pass an empty string if none is needed.
// fromPage and toPage are clamped to the source's actual page range.
func (d *Document) InsertPDFRange(src []byte, fromPage, toPage int, password string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	if len(src) == 0 {
		return errors.New("gomupdf: InsertPDFRange: empty source")
	}
	return d.b.insertPDFRange(src, fromPage, toPage, password)
}
