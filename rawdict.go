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
// Returns malloc'd NUL-terminated string or NULL+err.
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
                    // Emit fontname with spaces replaced by '_' so the
                    // record is safely split on whitespace.
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
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// Char is a single extracted glyph with its geometry.
type Char struct {
	Rune   rune           // the Unicode code point
	Origin geometry.Point // glyph origin (baseline start), PDF points
	BBox   geometry.Rect  // glyph bounding box, PDF points
}

// TextLine is a line of characters within a block.
type TextLine struct {
	BBox  geometry.Rect
	Chars []Char
}

// TextSpan groups consecutive chars sharing font and size (PyMuPDF span).
type TextSpan struct {
	Font string
	Size float64
	BBox geometry.Rect
	Text string
}

// RawBlock is a text block with its lines (rawdict: per-char) or spans (dict).
type RawBlock struct {
	BBox  geometry.Rect
	Lines []TextLine // populated by RawDict (per-char detail)
	Spans []TextSpan // populated by Dict (font/size runs)
}

// rawDictC calls the C helper and returns the raw text output.
func (p *Page) rawDictC() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_rawdict(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: rawdict: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

// parseRawDict parses the C output into RawBlock slices.
// Each B line starts a block, L a line, C a char.
func parseRawDict(s string) ([]RawBlock, error) {
	pf := func(v string) float64 { f, _ := strconv.ParseFloat(v, 64); return f }

	var blocks []RawBlock
	var curBlock *RawBlock
	var curLine *TextLine

	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case "B":
			if len(f) < 5 {
				continue
			}
			blocks = append(blocks, RawBlock{
				BBox: geometry.Rect{X0: pf(f[1]), Y0: pf(f[2]), X1: pf(f[3]), Y1: pf(f[4])},
			})
			curBlock = &blocks[len(blocks)-1]
			curLine = nil
		case "L":
			if curBlock == nil || len(f) < 5 {
				continue
			}
			curBlock.Lines = append(curBlock.Lines, TextLine{
				BBox: geometry.Rect{X0: pf(f[1]), Y0: pf(f[2]), X1: pf(f[3]), Y1: pf(f[4])},
			})
			curLine = &curBlock.Lines[len(curBlock.Lines)-1]
		case "C":
			// C rune ox oy x0 y0 x1 y1 size fontname
			if curLine == nil || len(f) < 10 {
				continue
			}
			cp, _ := strconv.Atoi(f[1])
			r, _ := utf8.DecodeRuneInString(string(rune(cp)))
			if r == utf8.RuneError {
				r = rune(cp)
			}
			ch := Char{
				Rune:   rune(cp),
				Origin: geometry.Point{X: pf(f[2]), Y: pf(f[3])},
				BBox:   geometry.Rect{X0: pf(f[4]), Y0: pf(f[5]), X1: pf(f[6]), Y1: pf(f[7])},
			}
			_ = r
			curLine.Chars = append(curLine.Chars, ch)
		}
	}
	return blocks, nil
}

// buildSpans converts RawBlocks (with Lines) into span-grouped RawBlocks.
// Consecutive chars in a line sharing (font, size) are merged into one TextSpan.
func buildSpans(raw []RawBlock, s string) []RawBlock {
	// We need font/size per char — re-parse from the raw text.
	type charMeta struct {
		font string
		size float64
	}
	// Collect all C records in order.
	var metas []charMeta
	for _, line := range strings.Split(s, "\n") {
		f := strings.Fields(line)
		if len(f) < 10 || f[0] != "C" {
			continue
		}
		size, _ := strconv.ParseFloat(f[8], 64)
		font := f[9]
		metas = append(metas, charMeta{font: font, size: size})
	}

	mi := 0
	out := make([]RawBlock, 0, len(raw))
	for _, rb := range raw {
		nb := RawBlock{BBox: rb.BBox}
		for _, tl := range rb.Lines {
			var curSpan *TextSpan
			for _, ch := range tl.Chars {
				var font string
				var size float64
				if mi < len(metas) {
					font = metas[mi].font
					size = metas[mi].size
					mi++
				}
				if curSpan == nil || curSpan.Font != font || curSpan.Size != size {
					nb.Spans = append(nb.Spans, TextSpan{
						Font: font,
						Size: size,
						BBox: ch.BBox,
						Text: string(ch.Rune),
					})
					curSpan = &nb.Spans[len(nb.Spans)-1]
				} else {
					curSpan.Text += string(ch.Rune)
					curSpan.BBox = curSpan.BBox.IncludeRect(ch.BBox)
				}
			}
		}
		out = append(out, nb)
	}
	return out
}

// RawDict returns the page's text blocks down to per-character geometry.
func (p *Page) RawDict() ([]RawBlock, error) {
	s, err := p.rawDictC()
	if err != nil {
		return nil, err
	}
	return parseRawDict(s)
}

// Dict returns the page's text blocks grouped into font/size spans.
func (p *Page) Dict() ([]RawBlock, error) {
	s, err := p.rawDictC()
	if err != nil {
		return nil, err
	}
	raw, err := parseRawDict(s)
	if err != nil {
		return nil, err
	}
	return buildSpans(raw, s), nil
}
