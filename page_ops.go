package gomupdf

/*
#cgo darwin CFLAGS: -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -lmupdf
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -lmupdf -lmupdf-third

#include <mupdf/fitz.h>
#include <mupdf/pdf.h>
#include <stdlib.h>
#include <string.h>

// gomupdf_copy_page inserts a shallow copy of page `from` at index `to`.
// Both indices are 0-based; `to` may equal the current page count (append).
// Returns 0 on success, -1 on error (message written to err).
static int gomupdf_copy_page(fz_context *ctx, fz_document *doc,
                             int from, int to, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) {
        pdf_obj *src = pdf_lookup_page_obj(ctx, pdf, from);
        pdf_insert_page(ctx, pdf, to, src);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// gomupdf_move_page moves page `from` to index `to` (0-based).
// After deletion the indices shift: if to > from, the effective insert point
// is (to - 1). The insert index is clamped to [0, count-1] after deletion.
static int gomupdf_move_page(fz_context *ctx, fz_document *doc,
                             int from, int to, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_obj *src = NULL;
    fz_var(src);
    fz_try(ctx) {
        src = pdf_keep_obj(ctx, pdf_lookup_page_obj(ctx, pdf, from));
        pdf_delete_page(ctx, pdf, from);
        // After deletion there are (n-1) pages.
        // If to > from the original destination shifted down by one.
        int n = pdf_count_pages(ctx, pdf); // n = old count - 1
        int ins = to;
        if (to > from) ins = to - 1;
        if (ins < 0) ins = 0;
        if (ins > n) ins = n;
        pdf_insert_page(ctx, pdf, ins, src);
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, src);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// gomupdf_select_pages rebuilds the document to contain only the pages listed
// in `pages` (0-based, length `nPages`), in order; indices may repeat.
// Fallback implementation (pdf_rearrange_pages not present in this MuPDF build).
static int gomupdf_select_pages(fz_context *ctx, fz_document *doc,
                                const int *pages, int nPages,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }

    pdf_obj **kept = NULL;
    fz_var(kept);
    fz_try(ctx) {
        // Keep a reference to each requested page object BEFORE any deletion.
        kept = (pdf_obj **)malloc(sizeof(pdf_obj *) * (size_t)nPages);
        if (!kept) fz_throw(ctx, FZ_ERROR_GENERIC, "out of memory");
        for (int i = 0; i < nPages; i++) kept[i] = NULL;
        for (int i = 0; i < nPages; i++)
            kept[i] = pdf_keep_obj(ctx, pdf_lookup_page_obj(ctx, pdf, pages[i]));

        // Delete all original pages.
        int orig = pdf_count_pages(ctx, pdf);
        for (int i = orig - 1; i >= 0; i--)
            pdf_delete_page(ctx, pdf, i);

        // Re-insert in desired order.
        for (int i = 0; i < nPages; i++)
            pdf_insert_page(ctx, pdf, i, kept[i]);
    }
    fz_always(ctx) {
        if (kept) {
            for (int i = 0; i < nPages; i++) pdf_drop_obj(ctx, kept[i]);
            free(kept);
        }
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// gomupdf_pagecount_pdf returns the page count of a pdf, or -1 on error.
static int gomupdf_pagecount_pdf(fz_context *ctx, fz_document *doc,
                                 char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { if (err) snprintf(err, errlen, "not a PDF document"); return -1; }
    int n = -1;
    fz_try(ctx) { n = pdf_count_pages(ctx, pdf); }
    fz_catch(ctx) {
        if (err) snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return n;
}

// gomupdf_insert_pdf_range appends pages [fromPage, toPage] (0-based, inclusive)
// of the source PDF (given as raw bytes) to the end of dst.
// password may be NULL or empty.
static int gomupdf_insert_pdf_range(fz_context *ctx, fz_document *dst_doc,
                                    const unsigned char *srcdata, size_t srclen,
                                    int fromPage, int toPage,
                                    const char *password,
                                    char *err, int errlen) {
    pdf_document *dst = pdf_specifics(ctx, dst_doc);
    if (!dst) { snprintf(err, errlen, "destination is not a PDF"); return -1; }

    fz_stream *stream = NULL;
    fz_document *srcfz = NULL;
    pdf_graft_map *map = NULL;
    fz_var(stream);
    fz_var(srcfz);
    fz_var(map);
    fz_try(ctx) {
        stream = fz_open_memory(ctx, srcdata, srclen);
        srcfz = fz_open_document_with_stream(ctx, ".pdf", stream);
        if (password && password[0] && fz_needs_password(ctx, srcfz))
            fz_authenticate_password(ctx, srcfz, password);
        pdf_document *src = pdf_specifics(ctx, srcfz);
        if (!src) fz_throw(ctx, FZ_ERROR_GENERIC, "source is not a PDF");

        int n = pdf_count_pages(ctx, src);
        // Clamp range.
        if (fromPage < 0) fromPage = 0;
        if (toPage >= n) toPage = n - 1;
        if (fromPage > toPage) fz_throw(ctx, FZ_ERROR_GENERIC, "empty page range");

        map = pdf_new_graft_map(ctx, dst);
        for (int i = fromPage; i <= toPage; i++)
            pdf_graft_mapped_page(ctx, map, -1, src, i);
    }
    fz_always(ctx) {
        pdf_drop_graft_map(ctx, map);
        fz_drop_document(ctx, srcfz);
        fz_drop_stream(ctx, stream);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// CopyPage inserts a shallow copy of page from (0-based) so the copy lands at
// index to (0-based). Use PageCount() as to in order to append at the end.
// The copy shares the source page's content stream (shallow reference),
// mirroring PyMuPDF's copy_page behaviour.
// Returns an error if either index is out of range or the document is closed.
func (d *Document) CopyPage(from, to int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	n := int(C.gomupdf_pagecount_pdf(d.ctx, d.doc, errBuf, errBufLen))
	if from < 0 || from >= n {
		return errors.New("gomupdf: CopyPage: from index out of range")
	}
	if to < 0 || to > n {
		return errors.New("gomupdf: CopyPage: to index out of range")
	}
	if C.gomupdf_copy_page(d.ctx, d.doc, C.int(from), C.int(to), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: CopyPage: " + C.GoString(errBuf))
	}
	return nil
}

// MovePage moves page from (0-based) to index to (0-based).
// After the move the page that was at from will be at to (considering the
// shift caused by the removal). Returns an error if either index is out of
// range or the document is closed.
func (d *Document) MovePage(from, to int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	n := int(C.gomupdf_pagecount_pdf(d.ctx, d.doc, errBuf, errBufLen))
	if from < 0 || from >= n {
		return errors.New("gomupdf: MovePage: from index out of range")
	}
	if to < 0 || to >= n {
		return errors.New("gomupdf: MovePage: to index out of range")
	}
	if C.gomupdf_move_page(d.ctx, d.doc, C.int(from), C.int(to), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: MovePage: " + C.GoString(errBuf))
	}
	return nil
}

// SelectPages rebuilds the document to contain only the given pages, in the
// given order (0-based indices; may repeat to duplicate pages). This is the
// fallback implementation — pdf_rearrange_pages is not present in this MuPDF
// build. Returns an error on invalid indices or a closed document.
func (d *Document) SelectPages(pages []int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	if len(pages) == 0 {
		return errors.New("gomupdf: SelectPages: pages list is empty")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	n := int(C.gomupdf_pagecount_pdf(d.ctx, d.doc, errBuf, errBufLen))
	for _, p := range pages {
		if p < 0 || p >= n {
			return errors.New("gomupdf: SelectPages: page index out of range")
		}
	}
	cPages := make([]C.int, len(pages))
	for i, p := range pages {
		cPages[i] = C.int(p)
	}
	if C.gomupdf_select_pages(d.ctx, d.doc,
		(*C.int)(unsafe.Pointer(&cPages[0])), C.int(len(cPages)),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: SelectPages: " + C.GoString(errBuf))
	}
	return nil
}

// InsertPDFRange appends pages [fromPage, toPage] (0-based, inclusive) of the
// source PDF (provided as raw bytes) to the end of this document. password
// unlocks an encrypted source; pass an empty string if none is needed.
// fromPage and toPage are clamped to the source's actual page range.
func (d *Document) InsertPDFRange(src []byte, fromPage, toPage int, password string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	if len(src) == 0 {
		return errors.New("gomupdf: InsertPDFRange: empty source")
	}
	cdata := C.CBytes(src)
	defer C.free(cdata)
	cpw := C.CString(password)
	defer C.free(unsafe.Pointer(cpw))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_insert_pdf_range(d.ctx, d.doc,
		(*C.uchar)(cdata), C.size_t(len(src)),
		C.int(fromPage), C.int(toPage),
		cpw, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: InsertPDFRange: " + C.GoString(errBuf))
	}
	return nil
}
