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

// apply_annot_style sets color, interior color, border width, and opacity on annot.
// stroke_rgb / fill_rgb are arrays of 3 floats; fill_rgb may be NULL (no fill).
static void apply_annot_style(fz_context *ctx, pdf_annot *annot,
                               float *stroke_rgb, float *fill_rgb,
                               float width, float opacity) {
    pdf_set_annot_color(ctx, annot, 3, stroke_rgb);
    if (fill_rgb) {
        pdf_set_annot_interior_color(ctx, annot, 3, fill_rgb);
    }
    pdf_set_annot_border_width(ctx, annot, width);
    pdf_set_annot_opacity(ctx, annot, opacity);
}

// gomupdf_add_line adds a PDF_ANNOT_LINE annotation between points a and b.
static int gomupdf_add_line(fz_context *ctx, fz_document *doc, int pageno,
                             float ax, float ay, float bx, float by,
                             float *stroke_rgb, float width, float opacity,
                             char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_var(page);
    fz_var(annot);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        annot = pdf_create_annot(ctx, page, PDF_ANNOT_LINE);
        fz_point pa = { ax, ay };
        fz_point pb = { bx, by };
        pdf_set_annot_line(ctx, annot, pa, pb);
        apply_annot_style(ctx, annot, stroke_rgb, NULL, width, opacity);
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

// gomupdf_add_circle adds a PDF_ANNOT_CIRCLE (ellipse inscribed in rect).
static int gomupdf_add_circle(fz_context *ctx, fz_document *doc, int pageno,
                               float x0, float y0, float x1, float y1,
                               float *stroke_rgb, float *fill_rgb,
                               float width, float opacity,
                               char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_var(page);
    fz_var(annot);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        annot = pdf_create_annot(ctx, page, PDF_ANNOT_CIRCLE);
        fz_rect r = { x0, y0, x1, y1 };
        pdf_set_annot_rect(ctx, annot, r);
        apply_annot_style(ctx, annot, stroke_rgb, fill_rgb, width, opacity);
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

// gomupdf_add_polygon adds a PDF_ANNOT_POLYGON (closed) with the given vertices.
// pts is a flat array of 2*npts floats: x0,y0,x1,y1,...
static int gomupdf_add_polygon(fz_context *ctx, fz_document *doc, int pageno,
                                float *pts, int npts,
                                float *stroke_rgb, float *fill_rgb,
                                float width, float opacity,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_var(page);
    fz_var(annot);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        annot = pdf_create_annot(ctx, page, PDF_ANNOT_POLYGON);
        for (int i = 0; i < npts; i++) {
            fz_point p = { pts[i*2], pts[i*2+1] };
            pdf_add_annot_vertex(ctx, annot, p);
        }
        apply_annot_style(ctx, annot, stroke_rgb, fill_rgb, width, opacity);
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

// gomupdf_add_polyline adds a PDF_ANNOT_POLY_LINE (open) with the given vertices.
static int gomupdf_add_polyline(fz_context *ctx, fz_document *doc, int pageno,
                                 float *pts, int npts,
                                 float *stroke_rgb, float width, float opacity,
                                 char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_var(page);
    fz_var(annot);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        annot = pdf_create_annot(ctx, page, PDF_ANNOT_POLY_LINE);
        for (int i = 0; i < npts; i++) {
            fz_point p = { pts[i*2], pts[i*2+1] };
            pdf_add_annot_vertex(ctx, annot, p);
        }
        apply_annot_style(ctx, annot, stroke_rgb, NULL, width, opacity);
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

// gomupdf_add_ink adds a PDF_ANNOT_INK (freehand) annotation.
// flat_pts: flat array of all points across all strokes (x0,y0,x1,y1,...).
// counts: per-stroke point count array of length nstrokes.
static int gomupdf_add_ink(fz_context *ctx, fz_document *doc, int pageno,
                            float *flat_pts, int *counts, int nstrokes,
                            float *stroke_rgb, float width, float opacity,
                            char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_point *fpts = NULL;
    fz_var(page);
    fz_var(annot);
    fz_var(fpts);
    fz_try(ctx) {
        // count total points
        int total = 0;
        for (int i = 0; i < nstrokes; i++) total += counts[i];
        fpts = (fz_point *)fz_malloc(ctx, total * sizeof(fz_point));
        for (int i = 0; i < total; i++) {
            fpts[i].x = flat_pts[i*2];
            fpts[i].y = flat_pts[i*2+1];
        }
        page = pdf_load_page(ctx, pdf, pageno);
        annot = pdf_create_annot(ctx, page, PDF_ANNOT_INK);
        pdf_set_annot_ink_list(ctx, annot, nstrokes, counts, fpts);
        apply_annot_style(ctx, annot, stroke_rgb, NULL, width, opacity);
        pdf_update_annot(ctx, annot);
    }
    fz_always(ctx) {
        fz_free(ctx, fpts);
        if (annot) pdf_drop_annot(ctx, annot);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// gomupdf_add_freetext adds a PDF_ANNOT_FREE_TEXT annotation.
static int gomupdf_add_freetext(fz_context *ctx, fz_document *doc, int pageno,
                                 float x0, float y0, float x1, float y1,
                                 const char *text, float size,
                                 float *stroke_rgb,
                                 char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_var(page);
    fz_var(annot);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        annot = pdf_create_annot(ctx, page, PDF_ANNOT_FREE_TEXT);
        fz_rect r = { x0, y0, x1, y1 };
        pdf_set_annot_rect(ctx, annot, r);
        pdf_set_annot_contents(ctx, annot, text);
        // Default appearance: Helvetica, given size, stroke color
        pdf_set_annot_default_appearance(ctx, annot, "Helv", size, 3, stroke_rgb);
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

// gomupdf_add_textnote adds a PDF_ANNOT_TEXT (sticky note) annotation.
static int gomupdf_add_textnote(fz_context *ctx, fz_document *doc, int pageno,
                                 float x, float y, const char *text,
                                 char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    pdf_annot *annot = NULL;
    fz_var(page);
    fz_var(annot);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        annot = pdf_create_annot(ctx, page, PDF_ANNOT_TEXT);
        // Place a 20x20 icon rect around the given point
        fz_rect r = { x, y, x + 20.0f, y + 20.0f };
        pdf_set_annot_rect(ctx, annot, r);
        pdf_set_annot_contents(ctx, annot, text);
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

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// AnnotStyle holds shared styling options for shape and text annotations.
// The zero value uses sensible defaults: black 1pt stroke, no fill, fully opaque.
type AnnotStyle struct {
	Stroke  *[3]float64 // RGB 0..1 border/line color; nil → black
	Fill    *[3]float64 // RGB 0..1 interior fill color; nil → no fill
	Width   float64     // border width in points; ≤0 → 1
	Opacity float64     // 0..1; ≤0 → 1 (fully opaque)
}

// resolveStyle applies defaults to an AnnotStyle and returns resolved values.
func resolveStyle(s AnnotStyle) (stroke [3]float64, hasFill bool, fill [3]float64, width, opacity float64) {
	if s.Stroke != nil {
		stroke = *s.Stroke
	} // else zero → black [0,0,0]
	if s.Fill != nil {
		hasFill = true
		fill = *s.Fill
	}
	width = s.Width
	if width <= 0 {
		width = 1
	}
	opacity = s.Opacity
	if opacity <= 0 {
		opacity = 1
	}
	return
}

// pointsToFlat converts a slice of geometry.Point into a flat []C.float [x0,y0,x1,y1,...].
func pointsToFlat(pts []geometry.Point) []C.float {
	flat := make([]C.float, len(pts)*2)
	for i, p := range pts {
		flat[i*2] = C.float(p.X)
		flat[i*2+1] = C.float(p.Y)
	}
	return flat
}

// AddLine adds a Line annotation from a to b on the page.
func (p *Page) AddLine(a, b geometry.Point, style AnnotStyle) error {
	stroke, _, _, width, opacity := resolveStyle(style)
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cStroke := [3]C.float{C.float(stroke[0]), C.float(stroke[1]), C.float(stroke[2])}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_line(d.ctx, d.doc, C.int(p.Number),
		C.float(a.X), C.float(a.Y), C.float(b.X), C.float(b.Y),
		&cStroke[0], C.float(width), C.float(opacity),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add line: " + C.GoString(errBuf))
	}
	return nil
}

// AddCircle adds a Circle (ellipse) annotation inscribed in rect r.
func (p *Page) AddCircle(r geometry.Rect, style AnnotStyle) error {
	stroke, hasFill, fill, width, opacity := resolveStyle(style)
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cStroke := [3]C.float{C.float(stroke[0]), C.float(stroke[1]), C.float(stroke[2])}
	var fillPtr *C.float
	var cFill [3]C.float
	if hasFill {
		cFill = [3]C.float{C.float(fill[0]), C.float(fill[1]), C.float(fill[2])}
		fillPtr = &cFill[0]
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_circle(d.ctx, d.doc, C.int(p.Number),
		C.float(r.X0), C.float(r.Y0), C.float(r.X1), C.float(r.Y1),
		&cStroke[0], fillPtr, C.float(width), C.float(opacity),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add circle: " + C.GoString(errBuf))
	}
	return nil
}

// AddPolygon adds a closed Polygon annotation with the given vertices.
func (p *Page) AddPolygon(pts []geometry.Point, style AnnotStyle) error {
	if len(pts) < 2 {
		return errors.New("gomupdf: polygon requires at least 2 points")
	}
	stroke, hasFill, fill, width, opacity := resolveStyle(style)
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	flat := pointsToFlat(pts)
	cStroke := [3]C.float{C.float(stroke[0]), C.float(stroke[1]), C.float(stroke[2])}
	var fillPtr *C.float
	var cFill [3]C.float
	if hasFill {
		cFill = [3]C.float{C.float(fill[0]), C.float(fill[1]), C.float(fill[2])}
		fillPtr = &cFill[0]
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_polygon(d.ctx, d.doc, C.int(p.Number),
		(*C.float)(unsafe.Pointer(&flat[0])), C.int(len(pts)),
		&cStroke[0], fillPtr, C.float(width), C.float(opacity),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add polygon: " + C.GoString(errBuf))
	}
	return nil
}

// AddPolyline adds an open PolyLine annotation with the given vertices.
func (p *Page) AddPolyline(pts []geometry.Point, style AnnotStyle) error {
	if len(pts) < 2 {
		return errors.New("gomupdf: polyline requires at least 2 points")
	}
	stroke, _, _, width, opacity := resolveStyle(style)
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	flat := pointsToFlat(pts)
	cStroke := [3]C.float{C.float(stroke[0]), C.float(stroke[1]), C.float(stroke[2])}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_polyline(d.ctx, d.doc, C.int(p.Number),
		(*C.float)(unsafe.Pointer(&flat[0])), C.int(len(pts)),
		&cStroke[0], C.float(width), C.float(opacity),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add polyline: " + C.GoString(errBuf))
	}
	return nil
}

// AddInk adds a freehand Ink annotation with one or more strokes.
func (p *Page) AddInk(strokes [][]geometry.Point, style AnnotStyle) error {
	if len(strokes) == 0 {
		return errors.New("gomupdf: ink requires at least one stroke")
	}
	stroke, _, _, width, opacity := resolveStyle(style)
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	// Flatten all strokes into one point array and build per-stroke counts.
	counts := make([]C.int, len(strokes))
	var allPts []geometry.Point
	for i, s := range strokes {
		counts[i] = C.int(len(s))
		allPts = append(allPts, s...)
	}
	flat := pointsToFlat(allPts)
	cStroke := [3]C.float{C.float(stroke[0]), C.float(stroke[1]), C.float(stroke[2])}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_ink(d.ctx, d.doc, C.int(p.Number),
		(*C.float)(unsafe.Pointer(&flat[0])),
		(*C.int)(unsafe.Pointer(&counts[0])), C.int(len(strokes)),
		&cStroke[0], C.float(width), C.float(opacity),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add ink: " + C.GoString(errBuf))
	}
	return nil
}

// AddFreeText adds a FreeText annotation with the given text and font size.
func (p *Page) AddFreeText(r geometry.Rect, text string, size float64, style AnnotStyle) error {
	stroke, _, _, _, _ := resolveStyle(style)
	if size <= 0 {
		size = 12
	}
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	cStroke := [3]C.float{C.float(stroke[0]), C.float(stroke[1]), C.float(stroke[2])}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_freetext(d.ctx, d.doc, C.int(p.Number),
		C.float(r.X0), C.float(r.Y0), C.float(r.X1), C.float(r.Y1),
		cText, C.float(size), &cStroke[0],
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add freetext: " + C.GoString(errBuf))
	}
	return nil
}

// AddTextNote adds a Text (sticky note) annotation at the given point.
func (p *Page) AddTextNote(at geometry.Point, text string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_textnote(d.ctx, d.doc, C.int(p.Number),
		C.float(at.X), C.float(at.Y), cText,
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add text note: " + C.GoString(errBuf))
	}
	return nil
}
