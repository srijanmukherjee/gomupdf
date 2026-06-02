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
// *out_len is set to total bytes. Returns NULL+err on failure.
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

        // Determine the pixel-integer bbox (possibly clipped).
        fz_rect r = fz_bound_page(ctx, page);
        if (have_clip) {
            fz_rect clip = fz_make_rect(clipx0, clipy0, clipx1, clipy1);
            r = fz_intersect_rect(r, clip);
        }
        fz_irect ir = fz_round_rect(fz_transform_rect(r, m));

        // Allocate pixmap and clear to white (or transparent).
        pix = fz_new_pixmap_with_bbox(ctx, cs, ir, NULL, alpha);
        if (alpha)
            fz_clear_pixmap(ctx, pix);
        else
            fz_clear_pixmap_with_value(ctx, pix, 0xff);

        // The draw device takes the render-to-pixmap matrix at creation time,
        // so we pass fz_identity to fz_run_page* -- the matrix is already baked
        // in. This is the canonical MuPDF pattern (also used by mutool draw).
        dev = fz_new_draw_device(ctx, m, pix);
        if (no_annots)
            fz_run_page_contents(ctx, page, dev, fz_identity, NULL);
        else
            fz_run_page(ctx, page, dev, fz_identity, NULL);
        fz_close_device(ctx, dev);
        fz_drop_device(ctx, dev);
        dev = NULL;

        // Pack header + samples into one malloc'd blob.
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
	"encoding/binary"
	"errors"
	"unsafe"
)

// renderRaw calls the richer gomupdf_render helper and returns the raw blob.
func (p *Page) renderRaw(zoom float64, csKind, alpha, haveClip int,
	clipx0, clipy0, clipx1, clipy1 float64, noAnnots int) ([]byte, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	var outLen C.int
	ptr := C.gomupdf_render(
		d.ctx, d.doc, C.int(p.Number),
		C.float(zoom), C.int(csKind), C.int(alpha),
		C.float(clipx0), C.float(clipy0), C.float(clipx1), C.float(clipy1),
		C.int(haveClip), C.int(noAnnots),
		&outLen, errBuf, errBufLen,
	)
	if ptr == nil {
		return nil, errors.New("gomupdf: render: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}

// parseBlob converts the 5×int32 header + samples blob into a Pixmap.
func parseBlob(blob []byte) (*Pixmap, error) {
	if len(blob) < 20 {
		return nil, errors.New("gomupdf: short pixmap blob")
	}
	rd := func(i int) int { return int(int32(binary.LittleEndian.Uint32(blob[i*4 : i*4+4]))) }
	// make a Go-owned copy of the samples so the caller owns the memory cleanly
	samples := make([]byte, len(blob)-20)
	copy(samples, blob[20:])
	return &Pixmap{
		Width:   rd(0),
		Height:  rd(1),
		N:       rd(2),
		Stride:  rd(3),
		Alpha:   rd(4) != 0,
		Samples: samples,
	}, nil
}

// Pixmap renders the page with the given options.
// Resolution precedence: DPI>0 → zoom=DPI/72; else Zoom (≤0 means 1).
// Colorspace precedence: CMYK → device CMYK; else Gray → device gray; else RGB.
func (p *Page) Pixmap(opts ...PixmapOptions) (*Pixmap, error) {
	o := PixmapOptions{Zoom: 1}
	if len(opts) > 0 {
		o = opts[0]
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
	csKind := 0
	if o.CMYK {
		csKind = 2
	} else if o.Gray {
		csKind = 1
	}

	alpha := 0
	if o.Alpha {
		alpha = 1
	}

	haveClip := 0
	var clipx0, clipy0, clipx1, clipy1 float64
	if o.Clip != nil {
		haveClip = 1
		clipx0, clipy0, clipx1, clipy1 = o.Clip.X0, o.Clip.Y0, o.Clip.X1, o.Clip.Y1
	}

	noAnnots := 0
	if o.NoAnnots {
		noAnnots = 1
	}

	blob, err := p.renderRaw(zoom, csKind, alpha, haveClip,
		clipx0, clipy0, clipx1, clipy1, noAnnots)
	if err != nil {
		return nil, err
	}
	return parseBlob(blob)
}
