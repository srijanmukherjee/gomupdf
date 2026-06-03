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

static char *gomupdf_list_annots(fz_context *ctx, fz_document *doc, int pageno,
                                 char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return NULL; }
    pdf_page *page = NULL;
    char *result = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        size_t cap = 4096;
        result = (char *)malloc(cap);
        if (!result) fz_throw(ctx, FZ_ERROR_GENERIC, "out of memory");
        result[0] = '\0';
        size_t used = 0;
        pdf_annot *a;
        for (a = pdf_first_annot(ctx, page); a; a = pdf_next_annot(ctx, a)) {
            enum pdf_annot_type t = pdf_annot_type(ctx, a);
            const char *tname = pdf_string_from_annot_type(ctx, t);
            fz_rect r = pdf_bound_annot(ctx, a);
            const char *contents = pdf_annot_contents(ctx, a);
            if (!contents) contents = "";
            char safe[512];
            size_t ci = 0;
            for (size_t j = 0; contents[j] && ci < sizeof(safe)-1; j++) {
                char c = contents[j];
                safe[ci++] = (c == '\t' || c == '\n' || c == '\r') ? ' ' : c;
            }
            safe[ci] = '\0';
            char line[1024];
            int n = snprintf(line, sizeof(line), "%s\t%g\t%g\t%g\t%g\t%s\n",
                             tname, r.x0, r.y0, r.x1, r.y1, safe);
            if (n < 0) n = 0;
            if (used + (size_t)n + 1 > cap) {
                cap = cap * 2 + (size_t)n + 1;
                char *tmp = (char *)realloc(result, cap);
                if (!tmp) fz_throw(ctx, FZ_ERROR_GENERIC, "out of memory");
                result = tmp;
            }
            memcpy(result + used, line, (size_t)n);
            used += (size_t)n;
            result[used] = '\0';
        }
    }
    fz_always(ctx) {
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

static int gomupdf_delete_annot(fz_context *ctx, fz_document *doc, int pageno,
                                int index, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_annot *a = pdf_first_annot(ctx, page);
        int i = 0;
        while (a && i < index) {
            a = pdf_next_annot(ctx, a);
            i++;
        }
        if (!a) fz_throw(ctx, FZ_ERROR_GENERIC, "annotation index out of range");
        pdf_delete_annot(ctx, page, a);
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

// annot_type: 0=highlight, 1=underline, 2=strikeout, 3=squiggly.
static int gomupdf_add_markup_annot(fz_context *ctx, fz_document *doc, int pageno,
                                    int annot_type, float *quads, int nquads,
                                    char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_var(page);
    fz_var(annot);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        enum pdf_annot_type t;
        switch (annot_type) {
            case 0: t = PDF_ANNOT_HIGHLIGHT; break;
            case 1: t = PDF_ANNOT_UNDERLINE; break;
            case 2: t = PDF_ANNOT_STRIKE_OUT; break;
            default: t = PDF_ANNOT_SQUIGGLY; break;
        }
        annot = pdf_create_annot(ctx, page, t);
        for (int i = 0; i < nquads; i++) {
            float *q = quads + i * 8;
            fz_quad fq;
            fq.ul.x = q[0]; fq.ul.y = q[1];
            fq.ur.x = q[2]; fq.ur.y = q[3];
            fq.ll.x = q[4]; fq.ll.y = q[5];
            fq.lr.x = q[6]; fq.lr.y = q[7];
            pdf_add_annot_quad_point(ctx, annot, fq);
        }
        float color[3];
        if (annot_type == 0) {
            color[0] = 1.0f; color[1] = 1.0f; color[2] = 0.0f;
        } else {
            color[0] = 1.0f; color[1] = 0.0f; color[2] = 0.0f;
        }
        pdf_set_annot_color(ctx, annot, 3, color);
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
*/
import "C"

import (
	"errors"
	"unsafe"
)

func (d *mupdfDoc) annotationsRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_list_annots(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: annotations: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) deleteAnnotation(pageNo, index int) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_delete_annot(d.ctx, d.doc, C.int(pageNo), C.int(index), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: delete annotation: " + C.GoString(errBuf))
	}
	return nil
}

// markupKinds maps the public kind string to the C annot_type int.
var markupKinds = map[string]int{
	"highlight": 0,
	"underline": 1,
	"strikeout": 2,
	"squiggly":  3,
}

func (d *mupdfDoc) addMarkup(pageNo int, kind string, quads []float32) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	annotType, ok := markupKinds[kind]
	if !ok {
		return errors.New("gomupdf: unknown markup kind: " + kind)
	}
	nq := len(quads) / 8
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	flat := make([]C.float, len(quads))
	for i, v := range quads {
		flat[i] = C.float(v)
	}
	if C.gomupdf_add_markup_annot(d.ctx, d.doc, C.int(pageNo), C.int(annotType),
		(*C.float)(unsafe.Pointer(&flat[0])), C.int(nq), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add markup annotation: " + C.GoString(errBuf))
	}
	return nil
}
