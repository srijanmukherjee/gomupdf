//go:build !nomupdf

package gomupdf

// Page content-writing helpers historically in mupdf.go and draw.go:
// insertText, insertImage, addRectAnnot, drawContent, drawText.

/*
#cgo darwin CFLAGS: -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -lmupdf
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -lmupdf -lmupdf-third

#include <mupdf/fitz.h>
#include <mupdf/pdf.h>
#include <stdlib.h>
#include <string.h>

// Insert a line of Helvetica text at (x,y) on a page (origin = baseline).
static int gomupdf_insert_text(fz_context *ctx, fz_document *doc, int pageno,
                               float x, float y, float size, const char *text,
                               char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_buffer *cbuf = NULL;
    pdf_obj *stream = NULL;
    fz_var(page);
    fz_var(cbuf);
    fz_var(stream);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_obj *pobj = page->obj;
        pdf_obj *res = pdf_dict_get(ctx, pobj, PDF_NAME(Resources));
        if (!res) {
            res = pdf_dict_put_dict(ctx, pobj, PDF_NAME(Resources), 2);
        }
        pdf_obj *fonts = pdf_dict_get(ctx, res, PDF_NAME(Font));
        if (!fonts) {
            fonts = pdf_dict_put_dict(ctx, res, PDF_NAME(Font), 1);
        }
        pdf_obj *font = pdf_new_dict(ctx, pdf, 4);
        pdf_dict_put(ctx, font, PDF_NAME(Type), PDF_NAME(Font));
        pdf_dict_put(ctx, font, PDF_NAME(Subtype), PDF_NAME(Type1));
        pdf_dict_put_drop(ctx, font, PDF_NAME(BaseFont), pdf_new_name(ctx, "Helvetica"));
        pdf_dict_puts_drop(ctx, fonts, "F0", pdf_add_object_drop(ctx, pdf, font));

        cbuf = fz_new_buffer(ctx, 128);
        fz_append_printf(ctx, cbuf, "\nq BT /F0 %g Tf %g %g Td (", size, x, y);
        for (const char *c = text; *c; c++) {
            if (*c == '(' || *c == ')' || *c == '\\') fz_append_byte(ctx, cbuf, '\\');
            fz_append_byte(ctx, cbuf, *c);
        }
        fz_append_printf(ctx, cbuf, ") Tj ET Q\n");

        stream = pdf_add_stream(ctx, pdf, cbuf, NULL, 0);
        pdf_obj *contents = pdf_dict_get(ctx, pobj, PDF_NAME(Contents));
        if (pdf_is_array(ctx, contents)) {
            pdf_array_push(ctx, contents, stream);
        } else {
            pdf_obj *arr = pdf_new_array(ctx, pdf, 2);
            if (contents) pdf_array_push(ctx, arr, contents);
            pdf_array_push(ctx, arr, stream);
            pdf_dict_put_drop(ctx, pobj, PDF_NAME(Contents), arr);
        }
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, stream);
        fz_drop_buffer(ctx, cbuf);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Insert an image (given as encoded bytes) into the rect (x,y,w,h) on a page.
static int gomupdf_insert_image(fz_context *ctx, fz_document *doc, int pageno,
                                float x, float y, float w, float h,
                                const unsigned char *imgdata, size_t imglen,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_buffer *ibuf = NULL;
    fz_image *img = NULL;
    pdf_obj *imgref = NULL;
    fz_buffer *cbuf = NULL;
    pdf_obj *stream = NULL;
    fz_var(page);
    fz_var(ibuf);
    fz_var(img);
    fz_var(imgref);
    fz_var(cbuf);
    fz_var(stream);
    fz_try(ctx) {
        ibuf = fz_new_buffer_from_copied_data(ctx, imgdata, imglen);
        img = fz_new_image_from_buffer(ctx, ibuf);
        imgref = pdf_add_image(ctx, pdf, img);

        page = pdf_load_page(ctx, pdf, pageno);
        pdf_obj *pobj = page->obj;
        pdf_obj *res = pdf_dict_get(ctx, pobj, PDF_NAME(Resources));
        if (!res) res = pdf_dict_put_dict(ctx, pobj, PDF_NAME(Resources), 2);
        pdf_obj *xobj = pdf_dict_get(ctx, res, PDF_NAME(XObject));
        if (!xobj) xobj = pdf_dict_put_dict(ctx, res, PDF_NAME(XObject), 1);
        char name[32];
        snprintf(name, sizeof(name), "GmImg%d", pdf_dict_len(ctx, xobj));
        pdf_dict_puts(ctx, xobj, name, imgref);

        cbuf = fz_new_buffer(ctx, 128);
        fz_append_printf(ctx, cbuf, "\nq %g 0 0 %g %g %g cm /%s Do Q\n", w, h, x, y, name);
        stream = pdf_add_stream(ctx, pdf, cbuf, NULL, 0);
        pdf_obj *contents = pdf_dict_get(ctx, pobj, PDF_NAME(Contents));
        if (pdf_is_array(ctx, contents)) {
            pdf_array_push(ctx, contents, stream);
        } else {
            pdf_obj *arr = pdf_new_array(ctx, pdf, 2);
            if (contents) pdf_array_push(ctx, arr, contents);
            pdf_array_push(ctx, arr, stream);
            pdf_dict_put_drop(ctx, pobj, PDF_NAME(Contents), arr);
        }
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, stream);
        fz_drop_buffer(ctx, cbuf);
        pdf_drop_obj(ctx, imgref);
        fz_drop_image(ctx, img);
        fz_drop_buffer(ctx, ibuf);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Add a rectangle annotation on a page.
static int gomupdf_add_rect_annot(fz_context *ctx, fz_document *doc, int pageno,
                                  float x0, float y0, float x1, float y1,
                                  char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_annot *annot = pdf_create_annot(ctx, page, PDF_ANNOT_SQUARE);
        fz_rect r = fz_make_rect(x0, y0, x1, y1);
        pdf_set_annot_rect(ctx, annot, r);
        pdf_update_annot(ctx, annot);
        pdf_drop_annot(ctx, annot);
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

// Append a generic PDF content-stream fragment (wrapped in q…Q) to the page.
static int gomupdf_draw_content(fz_context *ctx, fz_document *doc, int pageno,
                                const char *fragment, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_buffer *cbuf = NULL;
    pdf_obj *stream = NULL;
    fz_var(page);
    fz_var(cbuf);
    fz_var(stream);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_obj *pobj = page->obj;
        cbuf = fz_new_buffer(ctx, 256);
        fz_append_printf(ctx, cbuf, "\nq\n%s\nQ\n", fragment);
        stream = pdf_add_stream(ctx, pdf, cbuf, NULL, 0);
        pdf_obj *contents = pdf_dict_get(ctx, pobj, PDF_NAME(Contents));
        if (pdf_is_array(ctx, contents)) {
            pdf_array_push(ctx, contents, stream);
        } else {
            pdf_obj *arr = pdf_new_array(ctx, pdf, 2);
            if (contents) pdf_array_push(ctx, arr, contents);
            pdf_array_push(ctx, arr, stream);
            pdf_dict_put_drop(ctx, pobj, PDF_NAME(Contents), arr);
        }
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, stream);
        fz_drop_buffer(ctx, cbuf);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Ensure /Resources/Font/F0 = Helvetica on the page, then append a BT...ET
// text content stream.
static int gomupdf_draw_text(fz_context *ctx, fz_document *doc, int pageno,
                             const char *fragment, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_buffer *cbuf = NULL;
    pdf_obj *stream = NULL;
    fz_var(page);
    fz_var(cbuf);
    fz_var(stream);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_obj *pobj = page->obj;
        pdf_obj *res = pdf_dict_get(ctx, pobj, PDF_NAME(Resources));
        if (!res) {
            res = pdf_dict_put_dict(ctx, pobj, PDF_NAME(Resources), 2);
        }
        pdf_obj *fonts = pdf_dict_get(ctx, res, PDF_NAME(Font));
        if (!fonts) {
            fonts = pdf_dict_put_dict(ctx, res, PDF_NAME(Font), 1);
        }
        pdf_obj *font = pdf_new_dict(ctx, pdf, 4);
        pdf_dict_put(ctx, font, PDF_NAME(Type), PDF_NAME(Font));
        pdf_dict_put(ctx, font, PDF_NAME(Subtype), PDF_NAME(Type1));
        pdf_dict_put_drop(ctx, font, PDF_NAME(BaseFont), pdf_new_name(ctx, "Helvetica"));
        pdf_dict_puts_drop(ctx, fonts, "F0", pdf_add_object_drop(ctx, pdf, font));

        cbuf = fz_new_buffer(ctx, 512);
        fz_append_string(ctx, cbuf, fragment);
        stream = pdf_add_stream(ctx, pdf, cbuf, NULL, 0);
        pdf_obj *contents = pdf_dict_get(ctx, pobj, PDF_NAME(Contents));
        if (pdf_is_array(ctx, contents)) {
            pdf_array_push(ctx, contents, stream);
        } else {
            pdf_obj *arr = pdf_new_array(ctx, pdf, 2);
            if (contents) pdf_array_push(ctx, arr, contents);
            pdf_array_push(ctx, arr, stream);
            pdf_dict_put_drop(ctx, pobj, PDF_NAME(Contents), arr);
        }
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, stream);
        fz_drop_buffer(ctx, cbuf);
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

func (d *mupdfDoc) insertText(pageNo int, x, y, size float64, text string) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	ct := C.CString(text)
	defer C.free(unsafe.Pointer(ct))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_insert_text(d.ctx, d.doc, C.int(pageNo), C.float(x), C.float(y), C.float(size), ct, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: insert text: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) insertImage(pageNo int, rect [4]float64, img []byte) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cdata := C.CBytes(img)
	defer C.free(cdata)
	w := rect[2] - rect[0]
	h := rect[3] - rect[1]
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_insert_image(d.ctx, d.doc, C.int(pageNo),
		C.float(rect[0]), C.float(rect[1]), C.float(w), C.float(h),
		(*C.uchar)(cdata), C.size_t(len(img)), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: insert image: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) addRectAnnot(pageNo int, rect [4]float64) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_rect_annot(d.ctx, d.doc, C.int(pageNo),
		C.float(rect[0]), C.float(rect[1]), C.float(rect[2]), C.float(rect[3]), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add annot: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) drawContent(pageNo int, fragment string) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cf := C.CString(fragment)
	defer C.free(unsafe.Pointer(cf))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_draw_content(d.ctx, d.doc, C.int(pageNo), cf, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: draw content: " + C.GoString(errBuf))
	}
	return nil
}

func (d *mupdfDoc) drawText(pageNo int, fragment string) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cf := C.CString(fragment)
	defer C.free(unsafe.Pointer(cf))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_draw_text(d.ctx, d.doc, C.int(pageNo), cf, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: draw text: " + C.GoString(errBuf))
	}
	return nil
}
