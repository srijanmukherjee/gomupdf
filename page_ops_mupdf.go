//go:build !nomupdf

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

// Append a blank page of the given size at the end.
static int gomupdf_add_blank_page(fz_context *ctx, fz_document *doc, float w, float h,
                                  char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_buffer *contents = NULL;
    pdf_obj *resources = NULL;
    pdf_obj *page = NULL;
    fz_var(contents);
    fz_var(resources);
    fz_var(page);
    fz_try(ctx) {
        fz_rect mediabox = fz_make_rect(0, 0, w, h);
        contents = fz_new_buffer(ctx, 16);
        resources = pdf_new_dict(ctx, pdf, 1);
        page = pdf_add_page(ctx, pdf, mediabox, 0, resources, contents);
        pdf_insert_page(ctx, pdf, pdf_count_pages(ctx, pdf), page);
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, page);
        pdf_drop_obj(ctx, resources);
        fz_drop_buffer(ctx, contents);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

static int gomupdf_delete_page(fz_context *ctx, fz_document *doc, int n, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) { pdf_delete_page(ctx, pdf, n); }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Append all pages of a source PDF (given as bytes) to the end of dst.
static int gomupdf_graft_bytes(fz_context *ctx, fz_document *dst_doc,
                               const unsigned char *srcdata, size_t srclen,
                               const char *password, char *err, int errlen) {
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
        map = pdf_new_graft_map(ctx, dst);
        for (int i = 0; i < n; i++)
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

static int gomupdf_move_page(fz_context *ctx, fz_document *doc,
                             int from, int to, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_obj *src = NULL;
    fz_var(src);
    fz_try(ctx) {
        src = pdf_keep_obj(ctx, pdf_lookup_page_obj(ctx, pdf, from));
        pdf_delete_page(ctx, pdf, from);
        int n = pdf_count_pages(ctx, pdf);
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

static int gomupdf_select_pages(fz_context *ctx, fz_document *doc,
                                const int *pages, int nPages,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }

    pdf_obj **kept = NULL;
    fz_var(kept);
    fz_try(ctx) {
        kept = (pdf_obj **)malloc(sizeof(pdf_obj *) * (size_t)nPages);
        if (!kept) fz_throw(ctx, FZ_ERROR_GENERIC, "out of memory");
        for (int i = 0; i < nPages; i++) kept[i] = NULL;
        for (int i = 0; i < nPages; i++)
            kept[i] = pdf_keep_obj(ctx, pdf_lookup_page_obj(ctx, pdf, pages[i]));

        int orig = pdf_count_pages(ctx, pdf);
        for (int i = orig - 1; i >= 0; i--)
            pdf_delete_page(ctx, pdf, i);

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

func (d *mupdfDoc) newPage(width, height float64) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_blank_page(d.ctx, d.doc, C.float(width), C.float(height), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: new page: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) deletePage(n int) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_delete_page(d.ctx, d.doc, C.int(n), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: delete page: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) insertPDF(src []byte, password string) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cdata := C.CBytes(src)
	defer C.free(cdata)
	cpw := C.CString(password)
	defer C.free(unsafe.Pointer(cpw))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_graft_bytes(d.ctx, d.doc, (*C.uchar)(cdata), C.size_t(len(src)), cpw, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: insert pdf: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) pdfPageCount() (int, error) {
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	n := int(C.gomupdf_pagecount_pdf(d.ctx, d.doc, errBuf, errBufLen))
	if n < 0 {
		return 0, errors.New("gomupdf: " + C.GoString(errBuf))
	}
	return n, nil
}

func (d *mupdfDoc) copyPage(from, to int) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	n, err := d.pdfPageCount()
	if err != nil {
		return err
	}
	if from < 0 || from >= n {
		return errors.New("gomupdf: CopyPage: from index out of range")
	}
	if to < 0 || to > n {
		return errors.New("gomupdf: CopyPage: to index out of range")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_copy_page(d.ctx, d.doc, C.int(from), C.int(to), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: CopyPage: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) movePage(from, to int) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	n, err := d.pdfPageCount()
	if err != nil {
		return err
	}
	if from < 0 || from >= n {
		return errors.New("gomupdf: MovePage: from index out of range")
	}
	if to < 0 || to >= n {
		return errors.New("gomupdf: MovePage: to index out of range")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_move_page(d.ctx, d.doc, C.int(from), C.int(to), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: MovePage: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) selectPages(pages []int) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	n, err := d.pdfPageCount()
	if err != nil {
		return err
	}
	for _, p := range pages {
		if p < 0 || p >= n {
			return errors.New("gomupdf: SelectPages: page index out of range")
		}
	}
	cPages := make([]C.int, len(pages))
	for i, p := range pages {
		cPages[i] = C.int(p)
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_select_pages(d.ctx, d.doc,
		(*C.int)(unsafe.Pointer(&cPages[0])), C.int(len(cPages)),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: SelectPages: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) insertPDFRange(src []byte, fromPage, toPage int, password string) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
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
