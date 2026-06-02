package gomupdf

/*
#cgo darwin CFLAGS: -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -lmupdf
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -lmupdf -lmupdf-third

#include <mupdf/fitz.h>
#include <mupdf/pdf.h>
#include <stdlib.h>
#include <stdio.h>

// gomupdf_insert_link adds an external URI link over bbox on the given page.
// Returns 0 on success, -1 on error (errBuf filled).
static int gomupdf_insert_link(fz_context *ctx, fz_document *doc, int pageno,
                               float x0, float y0, float x1, float y1,
                               const char *uri, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        fz_rect bbox = fz_make_rect(x0, y0, x1, y1);
        pdf_create_link(ctx, page, bbox, uri);
    }
    fz_always(ctx) {
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// gomupdf_insert_goto_link adds an internal (goto) link that jumps to destPage
// (0-based). Uses fz_format_link_uri with FZ_LINK_DEST_FIT.
// Returns 0 on success, -1 on error.
static int gomupdf_insert_goto_link(fz_context *ctx, fz_document *doc, int pageno,
                                    float x0, float y0, float x1, float y1,
                                    int destPage, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    char *uri = NULL;
    fz_var(page);
    fz_var(uri);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        fz_rect bbox = fz_make_rect(x0, y0, x1, y1);
        fz_link_dest dest;
        dest.loc = fz_make_location(0, destPage);
        dest.type = FZ_LINK_DEST_FIT;
        dest.x = dest.y = dest.w = dest.h = dest.zoom = 0;
        uri = fz_format_link_uri(ctx, doc, dest);
        pdf_create_link(ctx, page, bbox, uri);
    }
    fz_always(ctx) {
        fz_free(ctx, uri);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// gomupdf_delete_link removes the link at 0-based index from the given page.
// Returns 0 on success, -1 on error.
static int gomupdf_delete_link(fz_context *ctx, fz_document *doc, int pageno,
                               int index, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_link *links = NULL;
    fz_var(page);
    fz_var(links);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        links = fz_load_links(ctx, (fz_page *)page);
        fz_link *cur = links;
        int i = 0;
        while (cur && i < index) { cur = cur->next; i++; }
        if (!cur) {
            snprintf(err, errlen, "link index %d out of range", index);
            fz_throw(ctx, FZ_ERROR_GENERIC, "link index out of range");
        }
        pdf_delete_link(ctx, page, cur);
    }
    fz_always(ctx) {
        fz_drop_link(ctx, links);
        fz_drop_page(ctx, (fz_page *)page);
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

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// InsertLink adds a clickable link over rect that opens an external URI.
// Changes take effect on the next Save.
func (p *Page) InsertLink(rect geometry.Rect, uri string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	curi := C.CString(uri)
	defer C.free(unsafe.Pointer(curi))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_insert_link(d.ctx, d.doc, C.int(p.Number),
		C.float(rect.X0), C.float(rect.Y0), C.float(rect.X1), C.float(rect.Y1),
		curi, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: insert link: " + C.GoString(errBuf))
	}
	return nil
}

// InsertGotoLink adds an internal link over rect that jumps to destPage (0-based).
// Changes take effect on the next Save.
func (p *Page) InsertGotoLink(rect geometry.Rect, destPage int) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_insert_goto_link(d.ctx, d.doc, C.int(p.Number),
		C.float(rect.X0), C.float(rect.Y0), C.float(rect.X1), C.float(rect.Y1),
		C.int(destPage), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: insert goto link: " + C.GoString(errBuf))
	}
	return nil
}

// DeleteLink removes the link at the given 0-based index (in Links() order).
// Changes take effect on the next Save.
func (p *Page) DeleteLink(index int) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_delete_link(d.ctx, d.doc, C.int(p.Number), C.int(index), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: delete link: " + C.GoString(errBuf))
	}
	return nil
}
