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

// gomupdf_list_annots returns a malloc'd string with one line per annotation:
//   Type\tx0\ty0\tx1\ty1\tContents
// Returns NULL + err on failure, or "" if no annotations.
static char *gomupdf_list_annots(fz_context *ctx, fz_document *doc, int pageno,
                                 char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return NULL; }
    pdf_page *page = NULL;
    char *result = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        // First pass: count and accumulate length
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
            // sanitize contents: replace tabs/newlines with spaces
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

// gomupdf_delete_annot deletes the annotation at 0-based index on the page.
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

// gomupdf_add_markup_annot adds a text-markup annotation (highlight/underline/
// strikeout/squiggly) on the given page. quads is a flat array of 8 floats per
// quad: ulx,uly,urx,ury,llx,lly,lrx,lry. nquads is the count of quads.
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
        // Colors: highlight=yellow {1,1,0}, others=red {1,0,0}
        float color[3];
        if (annot_type == 0) {
            color[0] = 1.0f; color[1] = 1.0f; color[2] = 0.0f; // yellow
        } else {
            color[0] = 1.0f; color[1] = 0.0f; color[2] = 0.0f; // red
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
	"strconv"
	"strings"
	"unsafe"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// Annotation describes an annotation present on a page.
type Annotation struct {
	Index    int           // 0-based position on the page (use with DeleteAnnotation)
	Type     string        // MuPDF annotation type name, e.g. "Highlight", "Square", "FreeText"
	Rect     geometry.Rect // bounding rect in PDF points
	Contents string        // text contents / note, if any
}

// Annotations returns the annotations on the page in document order.
func (p *Page) Annotations() ([]Annotation, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_list_annots(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return nil, errors.New("gomupdf: annotations: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	raw := C.GoString(cstr)
	if raw == "" {
		return nil, nil
	}
	var out []Annotation
	for i, ln := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if ln == "" {
			continue
		}
		// Format: Type\tx0\ty0\tx1\ty1\tContents
		parts := strings.SplitN(ln, "\t", 6)
		if len(parts) < 5 {
			continue
		}
		pf := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
		contents := ""
		if len(parts) == 6 {
			contents = parts[5]
		}
		out = append(out, Annotation{
			Index:    i,
			Type:     parts[0],
			Rect:     geometry.Rect{X0: pf(parts[1]), Y0: pf(parts[2]), X1: pf(parts[3]), Y1: pf(parts[4])},
			Contents: contents,
		})
	}
	return out, nil
}

// DeleteAnnotation removes the annotation at the given 0-based index.
func (p *Page) DeleteAnnotation(index int) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_delete_annot(d.ctx, d.doc, C.int(p.Number), C.int(index), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: delete annotation: " + C.GoString(errBuf))
	}
	return nil
}

// addMarkupAnnot is the shared implementation for highlight/underline/strikeout/squiggly.
// annotType: 0=highlight, 1=underline, 2=strikeout, 3=squiggly.
func (p *Page) addMarkupAnnot(annotType int, quads []geometry.Quad) error {
	if len(quads) == 0 {
		return errors.New("gomupdf: quads must not be empty")
	}
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	// Build flat float array: 8 floats per quad.
	flat := make([]C.float, len(quads)*8)
	for i, q := range quads {
		base := i * 8
		flat[base+0] = C.float(q.UL.X)
		flat[base+1] = C.float(q.UL.Y)
		flat[base+2] = C.float(q.UR.X)
		flat[base+3] = C.float(q.UR.Y)
		flat[base+4] = C.float(q.LL.X)
		flat[base+5] = C.float(q.LL.Y)
		flat[base+6] = C.float(q.LR.X)
		flat[base+7] = C.float(q.LR.Y)
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_markup_annot(d.ctx, d.doc, C.int(p.Number), C.int(annotType),
		(*C.float)(unsafe.Pointer(&flat[0])), C.int(len(quads)), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add markup annotation: " + C.GoString(errBuf))
	}
	return nil
}

// AddHighlight adds a highlight annotation covering the given quads.
// Color: yellow. Changes take effect on the next Save.
func (p *Page) AddHighlight(quads []geometry.Quad) error { return p.addMarkupAnnot(0, quads) }

// AddUnderline adds an underline annotation covering the given quads.
// Color: red. Changes take effect on the next Save.
func (p *Page) AddUnderline(quads []geometry.Quad) error { return p.addMarkupAnnot(1, quads) }

// AddStrikeout adds a strikeout annotation covering the given quads.
// Color: red. Changes take effect on the next Save.
func (p *Page) AddStrikeout(quads []geometry.Quad) error { return p.addMarkupAnnot(2, quads) }

// AddSquiggly adds a squiggly annotation covering the given quads.
// Color: red. Changes take effect on the next Save.
func (p *Page) AddSquiggly(quads []geometry.Quad) error { return p.addMarkupAnnot(3, quads) }
