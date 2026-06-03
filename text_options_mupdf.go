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

static char *gomupdf_page_text_opts(fz_context *ctx, fz_document *doc, int pageno,
                                    int flags, int clip_enabled,
                                    float cx0, float cy0, float cx1, float cy1,
                                    char *err, int errlen) {
    fz_page      *page  = NULL;
    fz_stext_page *stext = NULL;
    fz_buffer    *buf   = NULL;
    fz_output    *out   = NULL;
    char         *result = NULL;
    fz_var(page);
    fz_var(stext);
    fz_var(buf);
    fz_var(out);
    fz_try(ctx) {
        page = fz_load_page(ctx, doc, pageno);
        fz_stext_options opts;
        memset(&opts, 0, sizeof(opts));
        opts.flags = flags;
        stext = fz_new_stext_page_from_page(ctx, page, &opts);
        buf = fz_new_buffer(ctx, 4096);
        out = fz_new_output_with_buffer(ctx, buf);

        if (!clip_enabled) {
            fz_print_stext_page_as_text(ctx, out, stext);
        } else {
            fz_rect clip = fz_make_rect(cx0, cy0, cx1, cy1);
            for (fz_stext_block *block = stext->first_block; block; block = block->next) {
                if (block->type != FZ_STEXT_BLOCK_TEXT) continue;
                for (fz_stext_line *line = block->u.t.first_line; line; line = line->next) {
                    int wrote = 0;
                    for (fz_stext_char *ch = line->first_char; ch; ch = ch->next) {
                        fz_rect cr = fz_rect_from_quad(ch->quad);
                        fz_rect inter = fz_intersect_rect(cr, clip);
                        if (!fz_is_empty_rect(inter)) {
                            fz_write_rune(ctx, out, ch->c);
                            wrote = 1;
                        }
                    }
                    if (wrote) {
                        fz_write_byte(ctx, out, '\n');
                    }
                }
            }
        }

        fz_close_output(ctx, out);
        unsigned char *data = NULL;
        size_t n = fz_buffer_storage(ctx, buf, &data);
        result = (char *)malloc(n + 1);
        if (result) {
            memcpy(result, data, n);
            result[n] = '\0';
        }
    }
    fz_always(ctx) {
        fz_drop_output(ctx, out);
        fz_drop_buffer(ctx, buf);
        fz_drop_stext_page(ctx, stext);
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

func (d *mupdfDoc) extractTextRaw(pageNo int, opts TextOptions) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}

	var flags C.int
	if opts.PreserveLigatures {
		flags |= C.FZ_STEXT_PRESERVE_LIGATURES
	}
	if opts.PreserveWhitespace {
		flags |= C.FZ_STEXT_PRESERVE_WHITESPACE
	}
	if opts.PreserveImages {
		flags |= C.FZ_STEXT_PRESERVE_IMAGES
	}
	if opts.InhibitSpaces {
		flags |= C.FZ_STEXT_INHIBIT_SPACES
	}
	if opts.Dehyphenate {
		flags |= C.FZ_STEXT_DEHYPHENATE
	}

	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))

	var (
		clipEnabled C.int
		cx0, cy0    C.float
		cx1, cy1    C.float
	)
	if opts.Clip != nil {
		clipEnabled = 1
		cx0 = C.float(opts.Clip.X0)
		cy0 = C.float(opts.Clip.Y0)
		cx1 = C.float(opts.Clip.X1)
		cy1 = C.float(opts.Clip.Y1)
	}

	cstr := C.gomupdf_page_text_opts(d.ctx, d.doc, C.int(pageNo),
		flags, clipEnabled, cx0, cy0, cx1, cy1,
		errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: extract text: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}
