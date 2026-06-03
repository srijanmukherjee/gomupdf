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

static int gomupdf_set_metadata(fz_context *ctx, fz_document *doc, char *spec,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) {
        pdf_obj *trailer = pdf_trailer(ctx, pdf);
        if (!pdf_is_dict(ctx, pdf_resolve_indirect(ctx, pdf_dict_get(ctx, trailer, PDF_NAME(Info))))) {
            pdf_obj *ref = pdf_add_object_drop(ctx, pdf, pdf_new_dict(ctx, pdf, 8));
            pdf_dict_put(ctx, trailer, PDF_NAME(Info), ref);
            pdf_drop_obj(ctx, ref);
        }
        for (char *line = strtok(spec, "\n"); line; line = strtok(NULL, "\n")) {
            char *tab = strchr(line, '\t');
            if (!tab) continue;
            *tab = 0;
            char *field = line;
            char *value = tab + 1;
            char key[128];
            snprintf(key, sizeof(key), "info:%s", field);
            fz_set_metadata(ctx, doc, key, value);
        }
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

static unsigned char *gomupdf_get_xmp(fz_context *ctx, fz_document *doc,
                                      int *out_len, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return NULL; }
    fz_buffer *buf = NULL;
    unsigned char *result = NULL;
    *out_len = 0;
    fz_var(buf);
    fz_try(ctx) {
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
        pdf_obj *md = pdf_dict_get(ctx, root, PDF_NAME(Metadata));
        if (md && pdf_is_stream(ctx, md)) {
            buf = pdf_load_stream(ctx, md);
            unsigned char *data = NULL;
            size_t n = fz_buffer_storage(ctx, buf, &data);
            result = (unsigned char *)malloc(n ? n : 1);
            if (result) { memcpy(result, data, n); *out_len = (int)n; }
        }
    }
    fz_always(ctx) { fz_drop_buffer(ctx, buf); }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        *out_len = -1;
        return NULL;
    }
    return result;
}

static int gomupdf_set_xmp(fz_context *ctx, fz_document *doc,
                           const unsigned char *data, size_t len,
                           char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_buffer *buf = NULL;
    pdf_obj *dict = NULL;
    pdf_obj *ref = NULL;
    fz_var(buf);
    fz_var(dict);
    fz_var(ref);
    fz_try(ctx) {
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
        buf = fz_new_buffer_from_copied_data(ctx, data, len);
        dict = pdf_new_dict(ctx, pdf, 2);
        pdf_dict_put(ctx, dict, PDF_NAME(Type), PDF_NAME(Metadata));
        pdf_dict_put(ctx, dict, PDF_NAME(Subtype), PDF_NAME(XML));
        ref = pdf_add_object(ctx, pdf, dict);
        pdf_update_stream(ctx, pdf, ref, buf, 0);
        pdf_dict_put(ctx, root, PDF_NAME(Metadata), ref);
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, ref);
        pdf_drop_obj(ctx, dict);
        fz_drop_buffer(ctx, buf);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

static int gomupdf_del_xmp(fz_context *ctx, fz_document *doc, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) {
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
        pdf_dict_del(ctx, root, PDF_NAME(Metadata));
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

func (d *mupdfDoc) setMetadata(spec string) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cspec := C.CString(spec)
	defer C.free(unsafe.Pointer(cspec))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_set_metadata(d.ctx, d.doc, cspec, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set metadata: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) xmp() ([]byte, error) {
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	var outLen C.int
	ptr := C.gomupdf_get_xmp(d.ctx, d.doc, &outLen, errBuf, errBufLen)
	if ptr == nil {
		if outLen < 0 {
			return nil, errors.New("gomupdf: xmp: " + C.GoString(errBuf))
		}
		return nil, nil // no XMP stream
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}

func (d *mupdfDoc) setXMP(xml []byte) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cdata := C.CBytes(xml)
	defer C.free(cdata)
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_set_xmp(d.ctx, d.doc, (*C.uchar)(cdata), C.size_t(len(xml)), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set xmp: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) deleteXMP() error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_del_xmp(d.ctx, d.doc, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: delete xmp: " + C.GoString(errBuf))
	}
	return nil
}
