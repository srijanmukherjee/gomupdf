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
#include <stdio.h>

// gomupdf_rawdict walks the stext tree emitting one record per line:
//   B x0 y0 x1 y1          – block start (text blocks only)
//   L x0 y0 x1 y1          – line start
//   C rune ox oy x0 y0 x1 y1 size fontname  – character (fontname has no spaces)
static char *gomupdf_rawdict(fz_context *ctx, fz_document *doc, int pageno,
                             char *err, int errlen) {
    fz_page *page = NULL;
    fz_stext_page *stext = NULL;
    fz_buffer *buf = NULL;
    fz_output *out = NULL;
    char *result = NULL;
    fz_var(page);
    fz_var(stext);
    fz_var(buf);
    fz_var(out);
    fz_try(ctx) {
        page = fz_load_page(ctx, doc, pageno);
        fz_stext_options opts;
        memset(&opts, 0, sizeof(opts));
        stext = fz_new_stext_page_from_page(ctx, page, &opts);
        buf = fz_new_buffer(ctx, 16384);
        out = fz_new_output_with_buffer(ctx, buf);

        for (fz_stext_block *block = stext->first_block; block; block = block->next) {
            if (block->type != FZ_STEXT_BLOCK_TEXT) continue;
            fz_write_printf(ctx, out, "B %g %g %g %g\n",
                block->bbox.x0, block->bbox.y0, block->bbox.x1, block->bbox.y1);
            for (fz_stext_line *line = block->u.t.first_line; line; line = line->next) {
                fz_write_printf(ctx, out, "L %g %g %g %g\n",
                    line->bbox.x0, line->bbox.y0, line->bbox.x1, line->bbox.y1);
                for (fz_stext_char *ch = line->first_char; ch; ch = ch->next) {
                    fz_rect r = fz_rect_from_quad(ch->quad);
                    const char *fname = "";
                    if (ch->font)
                        fname = fz_font_name(ctx, ch->font);
                    fz_write_printf(ctx, out, "C %d %g %g %g %g %g %g %g ",
                        ch->c,
                        ch->origin.x, ch->origin.y,
                        r.x0, r.y0, r.x1, r.y1,
                        ch->size);
                    for (const char *p = fname; *p; p++) {
                        char c2 = (*p == ' ' || *p == '\t' || *p == '\n') ? '_' : *p;
                        fz_write_byte(ctx, out, (unsigned char)c2);
                    }
                    fz_write_byte(ctx, out, '\n');
                }
            }
        }

        fz_close_output(ctx, out);
        unsigned char *data = NULL;
        size_t n = fz_buffer_storage(ctx, buf, &data);
        result = (char *)malloc(n + 1);
        if (result) { memcpy(result, data, n); result[n] = '\0'; }
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

func (d *mupdfDoc) rawDictRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_rawdict(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: rawdict: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}
