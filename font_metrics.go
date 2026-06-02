package gomupdf

/*
#cgo darwin CFLAGS: -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -lmupdf
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -lmupdf -lmupdf-third

#include <mupdf/fitz.h>
#include <stdlib.h>
#include <string.h>

// Load a base-14 PDF font by name and return both the fz_font and the context
// that owns it. On error, writes to err and returns NULL (ctx is dropped).
static fz_font *gomupdf_new_base14(const char *name, fz_context **out_ctx,
                                    char *err, int errlen) {
    fz_context *ctx = fz_new_context(NULL, NULL, FZ_STORE_DEFAULT);
    if (!ctx) {
        snprintf(err, errlen, "gomupdf: failed to create context");
        return NULL;
    }
    fz_font *font = NULL;
    fz_try(ctx) {
        font = fz_new_base14_font(ctx, name);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        fz_drop_context(ctx);
        return NULL;
    }
    if (!font) {
        snprintf(err, errlen, "gomupdf: unknown base-14 font");
        fz_drop_context(ctx);
        return NULL;
    }
    *out_ctx = ctx;
    return font;
}

// Return the horizontal advance of a single unicode codepoint at 1 em
// (i.e. in em units). Returns 0 if the glyph is not found.
static float gomupdf_glyph_advance(fz_context *ctx, fz_font *font, int codepoint) {
    int gid = fz_encode_character(ctx, font, codepoint);
    if (gid <= 0) return 0.0f;
    return fz_advance_glyph(ctx, font, gid, 0);
}

// Fill *asc and *desc with the font's ascender and descender in em units.
static void gomupdf_font_vmetrics(fz_context *ctx, fz_font *font,
                                   float *asc, float *desc) {
    *asc  = fz_font_ascender(ctx, font);
    *desc = fz_font_descender(ctx, font);
}

// Return the font name (points into font internals; no need to free).
static const char *gomupdf_font_name_str(fz_context *ctx, fz_font *font) {
    return fz_font_name(ctx, font);
}
*/
import "C"

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

// Font is a loaded font usable for measuring text. Backed by a MuPDF font;
// call Close when done (a finalizer is a backstop).
type Font struct {
	mu   sync.Mutex
	ctx  *C.fz_context
	font *C.fz_font
}

// NewFont loads one of the 14 standard PDF fonts by name (e.g. "Helvetica",
// "Times-Roman", "Courier", "Helvetica-Bold", "Symbol", "ZapfDingbats").
func NewFont(name string) (*Font, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	errBuf := (*C.char)(C.malloc(C.size_t(errBufLen)))
	defer C.free(unsafe.Pointer(errBuf))

	var ctx *C.fz_context
	cfont := C.gomupdf_new_base14(cname, &ctx, errBuf, C.int(errBufLen))
	if cfont == nil {
		return nil, errors.New("gomupdf: NewFont: " + C.GoString(errBuf))
	}

	f := &Font{ctx: ctx, font: cfont}
	runtime.SetFinalizer(f, (*Font).Close)
	return f, nil
}

// TextLength returns the rendered width of text at the given font size (points).
func (f *Font) TextLength(text string, size float64) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.font == nil {
		return 0
	}
	var total float64
	for _, r := range text {
		adv := float64(C.gomupdf_glyph_advance(f.ctx, f.font, C.int(r)))
		total += adv
	}
	return total * size
}

// Advance returns the horizontal advance of a single rune at the given size.
func (f *Font) Advance(r rune, size float64) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.font == nil {
		return 0
	}
	adv := float64(C.gomupdf_glyph_advance(f.ctx, f.font, C.int(r)))
	return adv * size
}

// Ascender returns the font's ascender as a fraction of the em (×size for points).
func (f *Font) Ascender() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.font == nil {
		return 0
	}
	var asc, desc C.float
	C.gomupdf_font_vmetrics(f.ctx, f.font, &asc, &desc)
	return float64(asc)
}

// Descender returns the font's descender as a fraction of the em (negative).
func (f *Font) Descender() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.font == nil {
		return 0
	}
	var asc, desc C.float
	C.gomupdf_font_vmetrics(f.ctx, f.font, &asc, &desc)
	return float64(desc)
}

// Name returns the font's name.
func (f *Font) Name() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.font == nil {
		return ""
	}
	return C.GoString(C.gomupdf_font_name_str(f.ctx, f.font))
}

// Close releases the underlying font and context. Safe to call twice.
func (f *Font) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.font != nil {
		C.fz_drop_font(f.ctx, f.font)
		f.font = nil
	}
	if f.ctx != nil {
		C.fz_drop_context(f.ctx)
		f.ctx = nil
	}
	runtime.SetFinalizer(f, nil)
}
