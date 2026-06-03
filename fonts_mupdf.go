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

// gomupdf_page_fonts enumerates fonts in the page Resources/Font dict.
// Each font is one tab-separated line: xref TAB embedded TAB Subtype TAB Encoding TAB BaseFont
static char *gomupdf_page_fonts(fz_context *ctx, fz_document *doc, int pageno,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) {
        snprintf(err, errlen, "not a PDF document");
        return NULL;
    }

    pdf_page *page = NULL;
    fz_buffer *buf = NULL;
    char *result = NULL;
    fz_var(page);
    fz_var(buf);

    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_obj *res = pdf_page_resources(ctx, page);
        pdf_obj *fonts = res ? pdf_dict_get(ctx, res, PDF_NAME(Font)) : NULL;
        int n = fonts ? pdf_dict_len(ctx, fonts) : 0;

        buf = fz_new_buffer(ctx, 256);

        for (int i = 0; i < n; i++) {
            pdf_obj *fontref = pdf_dict_get_val(ctx, fonts, i);
            int xref = pdf_to_num(ctx, fontref);
            pdf_obj *fontobj = pdf_is_indirect(ctx, fontref)
                               ? pdf_resolve_indirect(ctx, fontref)
                               : fontref;

            const char *bfname = pdf_to_name(ctx, pdf_dict_get(ctx, fontobj, PDF_NAME(BaseFont)));
            if (!bfname) bfname = "";
            const char *subtype = pdf_to_name(ctx, pdf_dict_get(ctx, fontobj, PDF_NAME(Subtype)));
            if (!subtype) subtype = "";

            pdf_obj *encobj = pdf_dict_get(ctx, fontobj, PDF_NAME(Encoding));
            const char *enc = pdf_is_name(ctx, encobj) ? pdf_to_name(ctx, encobj) : "";
            if (!enc) enc = "";

            pdf_obj *desc = NULL;
            if (strcmp(subtype, "Type0") == 0) {
                pdf_obj *darr = pdf_dict_get(ctx, fontobj, PDF_NAME(DescendantFonts));
                if (pdf_is_array(ctx, darr) && pdf_array_len(ctx, darr) > 0) {
                    pdf_obj *df = pdf_array_get(ctx, darr, 0);
                    if (pdf_is_indirect(ctx, df))
                        df = pdf_resolve_indirect(ctx, df);
                    desc = pdf_dict_get(ctx, df, PDF_NAME(FontDescriptor));
                }
            } else {
                desc = pdf_dict_get(ctx, fontobj, PDF_NAME(FontDescriptor));
            }
            if (desc && pdf_is_indirect(ctx, desc))
                desc = pdf_resolve_indirect(ctx, desc);

            int embedded = 0;
            if (desc) {
                embedded = (pdf_dict_get(ctx, desc, PDF_NAME(FontFile))  != NULL) ||
                           (pdf_dict_get(ctx, desc, PDF_NAME(FontFile2)) != NULL) ||
                           (pdf_dict_get(ctx, desc, PDF_NAME(FontFile3)) != NULL);
            }

            fz_append_printf(ctx, buf, "%d\t%d\t%s\t%s\t%s\n",
                             xref, embedded, subtype, enc, bfname);
        }

        size_t sz = fz_buffer_storage(ctx, buf, NULL);
        result = (char *)malloc(sz + 1);
        if (result) {
            unsigned char *p = NULL;
            fz_buffer_storage(ctx, buf, &p);
            if (p)
                memcpy(result, p, sz);
            result[sz] = '\0';
        }
    }
    fz_always(ctx) {
        fz_drop_buffer(ctx, buf);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

// gomupdf_extract_font extracts the embedded font program for the PDF object at xref.
static unsigned char *gomupdf_extract_font(fz_context *ctx, fz_document *doc, int xref,
                                           int *out_len, char *ext_out, char *name_out,
                                           char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) {
        snprintf(err, errlen, "not a PDF document");
        return NULL;
    }

    unsigned char *result = NULL;
    pdf_obj *fontobj = NULL;
    fz_buffer *streambuf = NULL;
    int had_error = 0;
    fz_var(fontobj);
    fz_var(streambuf);

    ext_out[0] = '\0';
    name_out[0] = '\0';
    *out_len = 0;

    fz_try(ctx) {
        fontobj = pdf_load_object(ctx, pdf, xref);

        const char *bfname = pdf_to_name(ctx, pdf_dict_get(ctx, fontobj, PDF_NAME(BaseFont)));
        snprintf(name_out, 256, "%s", bfname ? bfname : "");

        const char *subtype = pdf_to_name(ctx, pdf_dict_get(ctx, fontobj, PDF_NAME(Subtype)));
        if (!subtype) subtype = "";

        pdf_obj *desc = NULL;
        if (strcmp(subtype, "Type0") == 0) {
            pdf_obj *darr = pdf_dict_get(ctx, fontobj, PDF_NAME(DescendantFonts));
            if (pdf_is_array(ctx, darr) && pdf_array_len(ctx, darr) > 0) {
                pdf_obj *df = pdf_array_get(ctx, darr, 0);
                if (pdf_is_indirect(ctx, df))
                    df = pdf_resolve_indirect(ctx, df);
                desc = pdf_dict_get(ctx, df, PDF_NAME(FontDescriptor));
            }
        } else {
            desc = pdf_dict_get(ctx, fontobj, PDF_NAME(FontDescriptor));
        }
        if (desc && pdf_is_indirect(ctx, desc))
            desc = pdf_resolve_indirect(ctx, desc);

        if (desc) {
            pdf_obj *stream = pdf_dict_get(ctx, desc, PDF_NAME(FontFile2));
            if (stream) {
                snprintf(ext_out, 8, "ttf");
            } else {
                stream = pdf_dict_get(ctx, desc, PDF_NAME(FontFile3));
                if (stream) {
                    pdf_obj *ffsubtype = pdf_dict_get(ctx, stream, PDF_NAME(Subtype));
                    const char *ffsub = pdf_to_name(ctx, ffsubtype);
                    if (ffsub && strcmp(ffsub, "OpenType") == 0)
                        snprintf(ext_out, 8, "otf");
                    else
                        snprintf(ext_out, 8, "cff");
                } else {
                    stream = pdf_dict_get(ctx, desc, PDF_NAME(FontFile));
                    if (stream)
                        snprintf(ext_out, 8, "pfa");
                }
            }

            if (stream) {
                streambuf = pdf_load_stream(ctx, stream);
                unsigned char *p = NULL;
                size_t sz = fz_buffer_storage(ctx, streambuf, &p);
                if (sz > 0 && p) {
                    result = (unsigned char *)malloc(sz);
                    if (result) {
                        memcpy(result, p, sz);
                        *out_len = (int)sz;
                    } else {
                        snprintf(err, errlen, "out of memory");
                        had_error = 1;
                    }
                }
            }
        }
    }
    fz_always(ctx) {
        fz_drop_buffer(ctx, streambuf);
        if (fontobj) pdf_drop_obj(ctx, fontobj);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        had_error = 1;
    }

    if (had_error && result == NULL)
        return NULL;
    return result;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func (d *mupdfDoc) fontsRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	*errBuf = 0
	cstr := C.gomupdf_page_fonts(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		msg := C.GoString(errBuf)
		if msg == "" {
			return "", nil
		}
		return "", errors.New("gomupdf: get fonts: " + msg)
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) extractFont(xref int) (name, ext string, data []byte, err error) {
	if d.doc == nil {
		return "", "", nil, errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	extBuf := (*C.char)(C.malloc(8))
	defer C.free(unsafe.Pointer(extBuf))
	nameBuf := (*C.char)(C.malloc(256))
	defer C.free(unsafe.Pointer(nameBuf))
	*errBuf = 0
	*extBuf = 0
	*nameBuf = 0

	var outLen C.int
	ptr := C.gomupdf_extract_font(d.ctx, d.doc, C.int(xref), &outLen, extBuf, nameBuf, errBuf, errBufLen)

	name = C.GoString(nameBuf)
	ext = C.GoString(extBuf)

	if ptr == nil {
		if msg := C.GoString(errBuf); msg != "" {
			return name, ext, nil, errors.New("gomupdf: extract font: " + msg)
		}
		return name, ext, nil, nil
	}
	defer C.free(unsafe.Pointer(ptr))
	return name, ext, C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}
