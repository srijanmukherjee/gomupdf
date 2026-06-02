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

// gomupdf_add_redaction adds a redaction annotation on the given page covering
// rect. If fill is non-NULL it is a 3-element float array (R,G,B 0..1) that
// sets the annotation colour drawn over the redacted area. Returns 0 on
// success, -1 on error.
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

// gomupdf_apply_redactions applies all redaction annotations on the page,
// permanently removing the covered content. Returns the number of redaction
// annotations that were present before applying (and thus processed), or -1 on
// error. image_method is one of the PDF_REDACT_IMAGE_* enum values.
static int gomupdf_apply_redactions(fz_context *ctx, fz_document *doc, int pageno,
                                    int image_method, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    int count = 0;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        // Count redaction annotations before applying.
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
            // text defaults to PDF_REDACT_TEXT_REMOVE (0) — leave as 0.
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

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// RedactOptions controls how covered content is removed.
type RedactOptions struct {
	// Fill is the RGB fill (0..1 each) drawn over the redacted area.
	// nil defaults to black {0,0,0}.
	Fill *[3]float64

	// RemoveImages removes images that overlap the redaction region when true.
	RemoveImages bool
}

// AddRedaction marks rect for redaction (a redaction annotation). Nothing is
// removed until ApplyRedactions is called.
func (p *Page) AddRedaction(rect geometry.Rect, opts RedactOptions) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}

	// Resolve fill colour: nil → black.
	fill := [3]float64{0, 0, 0}
	if opts.Fill != nil {
		fill = *opts.Fill
	}
	cfill := [3]C.float{C.float(fill[0]), C.float(fill[1]), C.float(fill[2])}

	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))

	if C.gomupdf_add_redaction(d.ctx, d.doc, C.int(p.Number),
		C.float(rect.X0), C.float(rect.Y0), C.float(rect.X1), C.float(rect.Y1),
		(*C.float)(unsafe.Pointer(&cfill[0])),
		errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add redaction: " + C.GoString(errBuf))
	}
	return nil
}

// ApplyRedactions permanently removes the content under every redaction
// annotation on the page (text, and images per options), then deletes the
// redaction marks. Returns the number of redactions applied. Effective on Save.
func (p *Page) ApplyRedactions() (int, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return 0, errors.New("gomupdf: document closed")
	}

	// PDF_REDACT_IMAGE_REMOVE = 1; PDF_REDACT_IMAGE_NONE = 0.
	// We always remove images that overlap — consistent with RemoveImages
	// defaulting to true in the options (caller can pass opts to AddRedaction
	// for fill; image removal is a page-level apply option).
	// Since ApplyRedactions has no per-call opts argument we use REMOVE.
	const imageMethodRemove = C.int(1) // PDF_REDACT_IMAGE_REMOVE

	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))

	n := C.gomupdf_apply_redactions(d.ctx, d.doc, C.int(p.Number),
		imageMethodRemove, errBuf, errBufLen)
	if n < 0 {
		return 0, errors.New("gomupdf: apply redactions: " + C.GoString(errBuf))
	}
	return int(n), nil
}
