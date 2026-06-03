//go:build !nomupdf

package gomupdf

// MuPDF backend (cgo). This file holds the core lifecycle helpers and the
// mupdfDoc/mupdfFont/mupdfDriver types implementing the package SPI in
// backend.go. The C helpers that were historically scattered across the
// per-operation files now live in matching <name>_mupdf.go files as methods on
// *mupdfDoc / *mupdfFont. cgo dedups C types across same-package files, so
// d.ctx (a *C.fz_context) created here is usable by C functions declared in any
// tagged file's preamble.

/*
#cgo darwin CFLAGS: -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -lmupdf
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -lmupdf -lmupdf-third

#include <mupdf/fitz.h>
#include <mupdf/pdf.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

// Create a context with document handlers registered. Returns NULL on failure.
static fz_context *gomupdf_new_context(void) {
    fz_context *ctx = fz_new_context(NULL, NULL, FZ_STORE_DEFAULT);
    if (!ctx) return NULL;
    fz_try(ctx) {
        fz_register_document_handlers(ctx);
    }
    fz_catch(ctx) {
        fz_drop_context(ctx);
        return NULL;
    }
    return ctx;
}

// Open a document from a memory buffer using an explicit magic/filetype hint
// (e.g. "png", ".pdf", "application/epub+zip"). The buffer must outlive the
// returned doc. On error returns NULL and writes the MuPDF message into err.
static fz_document *gomupdf_open_magic(fz_context *ctx, const unsigned char *data,
                                       size_t len, const char *magic,
                                       char *err, int errlen) {
    fz_document *doc = NULL;
    fz_stream *stream = NULL;
    fz_var(stream);
    fz_var(doc);
    fz_try(ctx) {
        stream = fz_open_memory(ctx, data, len);
        doc = fz_open_document_with_stream(ctx, magic, stream);
    }
    fz_always(ctx) {
        fz_drop_stream(ctx, stream);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return doc;
}

static int gomupdf_needs_password(fz_context *ctx, fz_document *doc) {
    int r = 0;
    fz_try(ctx) { r = fz_needs_password(ctx, doc); }
    fz_catch(ctx) { return 0; }
    return r;
}

// Returns 1 on success (or if no password needed), 0 on wrong password.
static int gomupdf_authenticate(fz_context *ctx, fz_document *doc, const char *pw) {
    int r = 0;
    fz_try(ctx) { r = fz_authenticate_password(ctx, doc, pw); }
    fz_catch(ctx) { return 0; }
    return r;
}

static int gomupdf_count_pages(fz_context *ctx, fz_document *doc, char *err, int errlen) {
    int n = -1;
    fz_try(ctx) { n = fz_count_pages(ctx, doc); }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return n;
}

// Create a new empty PDF, returned as an fz_document.
static fz_document *gomupdf_new_pdf(fz_context *ctx, char *err, int errlen) {
    pdf_document *pdf = NULL;
    fz_try(ctx) { pdf = pdf_create_document(ctx); }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return (fz_document *)pdf;
}

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
	"unsafe"
)

const errBufLen = 512

func init() {
	defaultDriver = mupdfDriver{}
}

// mupdfDriver is the MuPDF implementation of the driver SPI.
type mupdfDriver struct{}

func (mupdfDriver) name() string { return "mupdf" }

func (mupdfDriver) open(data []byte, magic string) (docBackend, bool, error) {
	if magic == "" {
		magic = ".pdf"
	}
	ctx := C.gomupdf_new_context()
	if ctx == nil {
		return nil, false, errors.New("gomupdf: failed to create context")
	}
	cdata := C.CBytes(data) // malloc'd copy; freed in close
	cmagic := C.CString(magic)
	defer C.free(unsafe.Pointer(cmagic))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))

	doc := C.gomupdf_open_magic(ctx, (*C.uchar)(cdata), C.size_t(len(data)), cmagic, errBuf, errBufLen)
	if doc == nil {
		C.free(cdata)
		C.fz_drop_context(ctx)
		return nil, false, errors.New("gomupdf: open failed: " + C.GoString(errBuf))
	}
	needsPass := C.gomupdf_needs_password(ctx, doc) != 0
	return &mupdfDoc{ctx: ctx, doc: doc, data: cdata, dataLen: len(data)}, needsPass, nil
}

func (mupdfDriver) newPDF() (docBackend, error) {
	ctx := C.gomupdf_new_context()
	if ctx == nil {
		return nil, errors.New("gomupdf: failed to create context")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	doc := C.gomupdf_new_pdf(ctx, errBuf, errBufLen)
	if doc == nil {
		C.fz_drop_context(ctx)
		return nil, errors.New("gomupdf: new pdf: " + C.GoString(errBuf))
	}
	return &mupdfDoc{ctx: ctx, doc: doc}, nil
}

func (mupdfDriver) newFont(name string) (fontBackend, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	errBuf := (*C.char)(C.malloc(C.size_t(errBufLen)))
	defer C.free(unsafe.Pointer(errBuf))
	var ctx *C.fz_context
	cfont := C.gomupdf_new_base14(cname, &ctx, errBuf, C.int(errBufLen))
	if cfont == nil {
		return nil, errors.New("gomupdf: NewFont: " + C.GoString(errBuf))
	}
	return &mupdfFont{ctx: ctx, font: cfont}, nil
}

// mupdfDoc is one open MuPDF document. Methods are NOT internally synchronized;
// the public Document serializes them under its mutex.
type mupdfDoc struct {
	ctx     *C.fz_context
	doc     *C.fz_document
	data    unsafe.Pointer // C-malloc'd copy backing the in-memory stream
	dataLen int            // length of the original bytes at data (incremental save)
}

func (d *mupdfDoc) close() {
	if d.doc != nil {
		C.fz_drop_document(d.ctx, d.doc)
		d.doc = nil
	}
	if d.ctx != nil {
		C.fz_drop_context(d.ctx)
		d.ctx = nil
	}
	if d.data != nil {
		C.free(d.data)
		d.data = nil
	}
}

func (d *mupdfDoc) authenticate(password string) bool {
	if d.doc == nil {
		return false
	}
	cpw := C.CString(password)
	defer C.free(unsafe.Pointer(cpw))
	return C.gomupdf_authenticate(d.ctx, d.doc, cpw) != 0
}

func (d *mupdfDoc) pageCount() int {
	if d.doc == nil {
		return 0
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	n := C.gomupdf_count_pages(d.ctx, d.doc, errBuf, errBufLen)
	if n < 0 {
		return 0
	}
	return int(n)
}

// mupdfFont is a loaded base-14 font owning its own context.
type mupdfFont struct {
	ctx  *C.fz_context
	font *C.fz_font
}

func (f *mupdfFont) close() {
	if f.font != nil {
		C.fz_drop_font(f.ctx, f.font)
		f.font = nil
	}
	if f.ctx != nil {
		C.fz_drop_context(f.ctx)
		f.ctx = nil
	}
}

func (f *mupdfFont) fontName() string {
	if f.font == nil {
		return ""
	}
	return C.GoString(C.gomupdf_font_name_str(f.ctx, f.font))
}

func (f *mupdfFont) advance(r rune) float64 {
	if f.font == nil {
		return 0
	}
	return float64(C.gomupdf_glyph_advance(f.ctx, f.font, C.int(r)))
}

func (f *mupdfFont) ascender() float64 {
	if f.font == nil {
		return 0
	}
	var asc, desc C.float
	C.gomupdf_font_vmetrics(f.ctx, f.font, &asc, &desc)
	return float64(asc)
}

func (f *mupdfFont) descender() float64 {
	if f.font == nil {
		return 0
	}
	var asc, desc C.float
	C.gomupdf_font_vmetrics(f.ctx, f.font, &asc, &desc)
	return float64(desc)
}

// ensure interface satisfaction
var (
	_ driver      = mupdfDriver{}
	_ docBackend  = (*mupdfDoc)(nil)
	_ fontBackend = (*mupdfFont)(nil)
)
