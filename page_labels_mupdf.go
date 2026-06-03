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
#include <stdio.h>

static int gomupdf_set_page_labels(fz_context *ctx, fz_document *doc,
                                   char *spec, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) {
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
        if (spec[0] == '\0') {
            pdf_dict_del(ctx, root, PDF_NAME(PageLabels));
        } else {
            pdf_obj *nums = pdf_new_array(ctx, pdf, 8);
            for (char *line = strtok(spec, "\n"); line; line = strtok(NULL, "\n")) {
                char *p = line;
                char *e1 = strchr(p, '\t'); if (!e1) continue; *e1 = 0;
                int startpage = atoi(p); p = e1 + 1;
                char *e2 = strchr(p, '\t'); if (!e2) continue; *e2 = 0;
                char *style = p; p = e2 + 1;
                char *e3 = strchr(p, '\t'); if (!e3) continue; *e3 = 0;
                int startnum = atoi(p); p = e3 + 1;
                char *prefix = p;
                pdf_array_push_int(ctx, nums, startpage);
                pdf_obj *ld = pdf_new_dict(ctx, pdf, 3);
                if (style[0]) pdf_dict_put_name(ctx, ld, PDF_NAME(S), style);
                if (prefix[0]) pdf_dict_put_text_string(ctx, ld, PDF_NAME(P), prefix);
                if (startnum > 1) pdf_dict_put_int(ctx, ld, PDF_NAME(St), startnum);
                pdf_array_push_drop(ctx, nums, ld);
            }
            pdf_obj *labels = pdf_new_dict(ctx, pdf, 1);
            pdf_dict_put_drop(ctx, labels, PDF_NAME(Nums), nums);
            pdf_dict_put_drop(ctx, root, PDF_NAME(PageLabels), pdf_add_object_drop(ctx, pdf, labels));
        }
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

static char *gomupdf_get_page_labels(fz_context *ctx, fz_document *doc,
                                     char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return NULL; }
    fz_buffer *buf = NULL;
    fz_output *out = NULL;
    char *result = NULL;
    fz_var(buf);
    fz_var(out);
    fz_try(ctx) {
        buf = fz_new_buffer(ctx, 256);
        out = fz_new_output_with_buffer(ctx, buf);
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
        pdf_obj *labels = pdf_dict_get(ctx, root, PDF_NAME(PageLabels));
        pdf_obj *nums = labels ? pdf_dict_get(ctx, labels, PDF_NAME(Nums)) : NULL;
        if (pdf_is_array(ctx, nums)) {
            int len = pdf_array_len(ctx, nums);
            for (int i = 0; i + 1 < len; i += 2) {
                int sp = pdf_to_int(ctx, pdf_array_get(ctx, nums, i));
                pdf_obj *d = pdf_array_get(ctx, nums, i + 1);
                pdf_obj *s = pdf_dict_get(ctx, d, PDF_NAME(S));
                const char *style = s ? pdf_to_name(ctx, s) : "";
                pdf_obj *pp = pdf_dict_get(ctx, d, PDF_NAME(P));
                const char *prefix = pp ? pdf_to_text_string(ctx, pp) : "";
                pdf_obj *st = pdf_dict_get(ctx, d, PDF_NAME(St));
                int start = st ? pdf_to_int(ctx, st) : 1;
                fz_write_printf(ctx, out, "%d\t%s\t%d\t%s\n", sp, style, start, prefix);
            }
        }
        fz_close_output(ctx, out);
        unsigned char *data = NULL;
        size_t sz = fz_buffer_storage(ctx, buf, &data);
        result = (char *)malloc(sz + 1);
        if (result) { memcpy(result, data, sz); result[sz] = '\0'; }
    }
    fz_always(ctx) {
        fz_drop_output(ctx, out);
        fz_drop_buffer(ctx, buf);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

static char *gomupdf_page_label(fz_context *ctx, fz_document *doc, int pageno,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    fz_page *page = NULL;
    char *result = NULL;
    fz_var(page);
    fz_try(ctx) {
        int have = 0;
        if (pdf) {
            pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
            have = pdf_dict_get(ctx, root, PDF_NAME(PageLabels)) != NULL;
        }
        result = (char *)malloc(128);
        if (!result) fz_throw(ctx, FZ_ERROR_GENERIC, "oom");
        result[0] = '\0';
        if (have) {
            page = fz_load_page(ctx, doc, pageno);
            fz_page_label(ctx, page, result, 128);
        }
    }
    fz_always(ctx) { fz_drop_page(ctx, page); }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func (d *mupdfDoc) setPageLabels(spec string) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cspec := C.CString(spec)
	defer C.free(unsafe.Pointer(cspec))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_set_page_labels(d.ctx, d.doc, cspec, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set page labels: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) pageLabelsRaw() (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_get_page_labels(d.ctx, d.doc, errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: page labels: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) pageLabel(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_label(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: page label: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}
