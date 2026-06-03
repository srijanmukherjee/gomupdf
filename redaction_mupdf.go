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
#include <stdio.h>
#include <string.h>

static int gomupdf_add_redaction(fz_context *ctx, fz_document *doc, int pageno,
                                 float x0, float y0, float x1, float y1,
                                 float *fill, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_var(page);
    fz_var(annot);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        annot = pdf_create_annot(ctx, page, PDF_ANNOT_REDACT);
        fz_rect r = fz_make_rect(x0, y0, x1, y1);
        pdf_set_annot_rect(ctx, annot, r);
        if (fill) {
            pdf_set_annot_color(ctx, annot, 3, fill);
        }
        pdf_update_annot(ctx, annot);
    }
    fz_always(ctx) {
        if (annot) pdf_drop_annot(ctx, annot);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

static int gomupdf_apply_redactions(fz_context *ctx, fz_document *doc, int pageno,
                                    int image_method, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    int count = 0;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_annot *a;
        for (a = pdf_first_annot(ctx, page); a; a = pdf_next_annot(ctx, a)) {
            if (pdf_annot_type(ctx, a) == PDF_ANNOT_REDACT) {
                count++;
            }
        }
        if (count > 0) {
            pdf_redact_options opts;
            memset(&opts, 0, sizeof(opts));
            opts.image_method = image_method;
            pdf_redact_page(ctx, pdf, page, &opts);
        }
    }
    fz_always(ctx) {
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return count;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func (d *mupdfDoc) addRedaction(pageNo int, rect [4]float64, fill *[3]float64) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	f := [3]float64{0, 0, 0}
	if fill != nil {
		f = *fill
	}
	cfill := [3]C.float{C.float(f[0]), C.float(f[1]), C.float(f[2])}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_redaction(d.ctx, d.doc, C.int(pageNo),
		C.float(rect[0]), C.float(rect[1]), C.float(rect[2]), C.float(rect[3]),
		(*C.float)(unsafe.Pointer(&cfill[0])),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add redaction: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) applyRedactions(pageNo int) (int, error) {
	if d.doc == nil {
		return 0, errors.New("gomupdf: document closed")
	}
	// PDF_REDACT_IMAGE_REMOVE = 1.
	const imageMethodRemove = C.int(1)
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	n := C.gomupdf_apply_redactions(d.ctx, d.doc, C.int(pageNo), imageMethodRemove, errBuf, errBufLen)
	if n < 0 {
		return 0, errors.New("gomupdf: apply redactions: " + C.GoString(errBuf))
	}
	return int(n), nil
}
