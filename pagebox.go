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

// Emit a page's geometry as "rot mx0 my0 mx1 my1 cx0 cy0 cx1 cy1 hascrop".
// Boxes are the raw, inheritable dictionary values in unrotated PDF points; the
// CropBox falls back to the MediaBox when absent (hascrop reports which).
static char *gomupdf_page_geometry(fz_context *ctx, fz_document *doc, int pageno,
                                   char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return NULL; }
    pdf_page *page = NULL;
    char *result = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_obj *obj = page->obj;
        int rot = pdf_to_int(ctx, pdf_dict_get_inheritable(ctx, obj, PDF_NAME(Rotate)));
        pdf_obj *mbo = pdf_dict_get_inheritable(ctx, obj, PDF_NAME(MediaBox));
        fz_rect mb = pdf_to_rect(ctx, mbo);
        pdf_obj *cbo = pdf_dict_get_inheritable(ctx, obj, PDF_NAME(CropBox));
        int hascrop = (cbo != NULL);
        fz_rect cb = hascrop ? fz_intersect_rect(pdf_to_rect(ctx, cbo), mb) : mb;
        result = (char *)malloc(256);
        if (result)
            snprintf(result, 256, "%d %g %g %g %g %g %g %g %g %d",
                     rot, mb.x0, mb.y0, mb.x1, mb.y1,
                     cb.x0, cb.y0, cb.x1, cb.y1, hascrop);
    }
    fz_always(ctx) { fz_drop_page(ctx, (fz_page *)page); }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

static int gomupdf_set_rotation(fz_context *ctx, fz_document *doc, int pageno,
                                int deg, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_dict_put_int(ctx, page->obj, PDF_NAME(Rotate), deg);
    }
    fz_always(ctx) { fz_drop_page(ctx, (fz_page *)page); }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// which: 0 = MediaBox, 1 = CropBox.
static int gomupdf_set_box(fz_context *ctx, fz_document *doc, int pageno, int which,
                           float x0, float y0, float x1, float y1,
                           char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        fz_rect r = fz_make_rect(x0, y0, x1, y1);
        pdf_obj *key = which == 1 ? PDF_NAME(CropBox) : PDF_NAME(MediaBox);
        pdf_dict_put_rect(ctx, page->obj, key, r);
    }
    fz_always(ctx) { fz_drop_page(ctx, (fz_page *)page); }
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
	"strconv"
	"strings"
	"unsafe"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// Page geometry: rotation and the /MediaBox and /CropBox page boxes.
//
// PDF defines several page boxes; gomupdf exposes the two that matter for
// rendering and layout: the MediaBox (the full physical page) and the CropBox
// (the visible region, defaulting to the MediaBox when absent). All boxes are
// reported in unrotated PDF points (origin bottom-left); use Rotation to read
// the display rotation separately. These mirror PyMuPDF's page.rotation /
// page.mediabox / page.cropbox.

// pageGeometry reads rotation, MediaBox, and CropBox in one cgo round-trip.
func (p *Page) pageGeometry() (rot int, media, crop geometry.Rect, hasCrop bool, err error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return 0, geometry.Rect{}, geometry.Rect{}, false, errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_geometry(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return 0, geometry.Rect{}, geometry.Rect{}, false, errors.New("gomupdf: page geometry: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	f := strings.Fields(C.GoString(cstr))
	if len(f) != 10 {
		return 0, geometry.Rect{}, geometry.Rect{}, false, errors.New("gomupdf: bad page geometry output")
	}
	rot, _ = strconv.Atoi(f[0])
	pf := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	media = geometry.Rect{X0: pf(f[1]), Y0: pf(f[2]), X1: pf(f[3]), Y1: pf(f[4])}
	crop = geometry.Rect{X0: pf(f[5]), Y0: pf(f[6]), X1: pf(f[7]), Y1: pf(f[8])}
	hasCrop = f[9] == "1"
	return rot, media, crop, hasCrop, nil
}

// normalizeAngle reduces deg into [0, 360).
func normalizeAngle(deg int) int {
	deg %= 360
	if deg < 0 {
		deg += 360
	}
	return deg
}

// Rotation returns the page's display rotation in degrees: one of 0, 90, 180,
// or 270. The value is normalized into [0, 360).
func (p *Page) Rotation() (int, error) {
	rot, _, _, _, err := p.pageGeometry()
	if err != nil {
		return 0, err
	}
	return normalizeAngle(rot), nil
}

// SetRotation sets the page's display rotation. deg must be a multiple of 90;
// it is normalized into [0, 360). Changes take effect on the next Save.
func (p *Page) SetRotation(deg int) error {
	if deg%90 != 0 {
		return errors.New("gomupdf: rotation must be a multiple of 90")
	}
	deg = normalizeAngle(deg)
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_set_rotation(d.ctx, d.doc, C.int(p.Number), C.int(deg), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set rotation: " + C.GoString(errBuf))
	}
	return nil
}

// MediaBox returns the page's /MediaBox — the full physical page rectangle in
// unrotated PDF points.
func (p *Page) MediaBox() (geometry.Rect, error) {
	_, media, _, _, err := p.pageGeometry()
	return media, err
}

// CropBox returns the page's /CropBox — the visible region in unrotated PDF
// points. When the page declares no CropBox, the MediaBox is returned.
func (p *Page) CropBox() (geometry.Rect, error) {
	_, _, crop, _, err := p.pageGeometry()
	return crop, err
}

// setBox is the shared cgo entry point for SetMediaBox/SetCropBox.
func (p *Page) setBox(which int, r geometry.Rect) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_set_box(d.ctx, d.doc, C.int(p.Number), C.int(which),
		C.float(r.X0), C.float(r.Y0), C.float(r.X1), C.float(r.Y1), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set box: " + C.GoString(errBuf))
	}
	return nil
}

// SetMediaBox sets the page's /MediaBox. The rectangle is given in unrotated
// PDF points. Changes take effect on the next Save.
func (p *Page) SetMediaBox(r geometry.Rect) error { return p.setBox(0, r) }

// SetCropBox sets the page's /CropBox. The rectangle is given in unrotated PDF
// points and is clamped to the MediaBox by MuPDF. Changes take effect on the
// next Save.
func (p *Page) SetCropBox(r geometry.Rect) error { return p.setBox(1, r) }
