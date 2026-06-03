//go:build !nomupdf

package gomupdf

/*
#cgo darwin CFLAGS: -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -lmupdf
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -lmupdf -lmupdf-third

#include <mupdf/fitz.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

// cs_kind: 0=rgb, 1=gray, 2=cmyk
// have_clip: 0=no clip, 1=use clipx0/y0/x1/y1
// no_annots: 0=render everything, 1=skip annotations
// Returns malloc'd blob: 5x int32 header [w,h,n,stride,alpha] + raw samples.
static unsigned char *gomupdf_render(
        fz_context *ctx, fz_document *doc, int pageno,
        float zoom, int cs_kind, int alpha,
        float clipx0, float clipy0, float clipx1, float clipy1,
        int have_clip, int no_annots,
        int *out_len, char *err, int errlen)
{
    fz_page    *page   = NULL;
    fz_pixmap  *pix    = NULL;
    fz_device  *dev    = NULL;
    unsigned char *result = NULL;

    fz_var(page);
    fz_var(pix);
    fz_var(dev);

    fz_try(ctx) {
        fz_matrix m = fz_scale(zoom, zoom);

        fz_colorspace *cs;
        switch (cs_kind) {
        case 1:  cs = fz_device_gray(ctx); break;
        case 2:  cs = fz_device_cmyk(ctx); break;
        default: cs = fz_device_rgb(ctx);  break;
        }

        page = fz_load_page(ctx, doc, pageno);

        fz_rect r = fz_bound_page(ctx, page);
        if (have_clip) {
            fz_rect clip = fz_make_rect(clipx0, clipy0, clipx1, clipy1);
            r = fz_intersect_rect(r, clip);
        }
        fz_irect ir = fz_round_rect(fz_transform_rect(r, m));

        pix = fz_new_pixmap_with_bbox(ctx, cs, ir, NULL, alpha);
        if (alpha)
            fz_clear_pixmap(ctx, pix);
        else
            fz_clear_pixmap_with_value(ctx, pix, 0xff);

        dev = fz_new_draw_device(ctx, m, pix);
        if (no_annots)
            fz_run_page_contents(ctx, page, dev, fz_identity, NULL);
        else
            fz_run_page(ctx, page, dev, fz_identity, NULL);
        fz_close_device(ctx, dev);
        fz_drop_device(ctx, dev);
        dev = NULL;

        int32_t hdr[5];
        hdr[0] = pix->w;
        hdr[1] = pix->h;
        hdr[2] = pix->n;
        hdr[3] = pix->stride;
        hdr[4] = pix->alpha;
        size_t samples = (size_t)pix->stride * pix->h;
        size_t total   = sizeof(hdr) + samples;
        result = (unsigned char *)malloc(total);
        if (result) {
            memcpy(result, hdr, sizeof(hdr));
            memcpy(result + sizeof(hdr), pix->samples, samples);
            *out_len = (int)total;
        }
    }
    fz_always(ctx) {
        fz_drop_device(ctx, dev);
        fz_drop_pixmap(ctx, pix);
        fz_drop_page(ctx, page);
    }
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

func (d *mupdfDoc) pixmap(pageNo int, o PixmapOptions) ([]byte, error) {
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}

	// Resolve zoom.
	zoom := o.Zoom
	if o.DPI > 0 {
		zoom = o.DPI / 72.0
	}
	if zoom <= 0 {
		zoom = 1
	}

	// Resolve colorspace kind: 0=rgb, 1=gray, 2=cmyk.
	csKind := C.int(0)
	if o.CMYK {
		csKind = 2
	} else if o.Gray {
		csKind = 1
	}

	alpha := C.int(0)
	if o.Alpha {
		alpha = 1
	}

	haveClip := C.int(0)
	var clipx0, clipy0, clipx1, clipy1 float64
	if o.Clip != nil {
		haveClip = 1
		clipx0, clipy0, clipx1, clipy1 = o.Clip.X0, o.Clip.Y0, o.Clip.X1, o.Clip.Y1
	}

	noAnnots := C.int(0)
	if o.NoAnnots {
		noAnnots = 1
	}

	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	var outLen C.int
	ptr := C.gomupdf_render(
		d.ctx, d.doc, C.int(pageNo),
		C.float(zoom), csKind, alpha,
		C.float(clipx0), C.float(clipy0), C.float(clipx1), C.float(clipy1),
		haveClip, noAnnots,
		&outLen, errBuf, errBufLen,
	)
	if ptr == nil {
		return nil, errors.New("gomupdf: render: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}
