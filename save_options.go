package gomupdf

/*
#cgo darwin CFLAGS: -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -lmupdf
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -lmupdf -lmupdf-third

#include <mupdf/fitz.h>
#include <mupdf/pdf.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

// Serialize a document with explicit pdf_write_options. For an incremental
// save (incremental != 0), origdata/origlen must hold the original file bytes;
// they are written first and the update section is appended after them.
static unsigned char *gomupdf_save_opts(
        fz_context *ctx, fz_document *doc,
        int garbage, int deflate, int deflate_images, int deflate_fonts,
        int clean, int linear, int ascii, int pretty,
        int encrypt, const char *upwd, const char *opwd, int permissions,
        int incremental, const unsigned char *origdata, size_t origlen,
        int *out_len, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return NULL; }
    fz_buffer *buf = NULL;
    fz_output *out = NULL;
    unsigned char *result = NULL;
    fz_var(buf);
    fz_var(out);
    fz_try(ctx) {
        pdf_write_options opts;
        memset(&opts, 0, sizeof(opts));
        if (garbage < 0) garbage = 0;
        if (garbage > 4) garbage = 4;
        opts.do_garbage = garbage;
        opts.do_compress = deflate ? 1 : 0;
        opts.do_compress_images = deflate_images ? 1 : 0;
        opts.do_compress_fonts = deflate_fonts ? 1 : 0;
        opts.do_clean = clean ? 1 : 0;
        opts.do_sanitize = clean ? 1 : 0;
        opts.do_linear = linear ? 1 : 0;
        opts.do_ascii = ascii ? 1 : 0;
        opts.do_pretty = pretty ? 1 : 0;
        if (encrypt) {
            opts.do_encrypt = PDF_ENCRYPT_AES_256;
            opts.permissions = permissions;
            snprintf(opts.upwd_utf8, sizeof(opts.upwd_utf8), "%s", upwd ? upwd : "");
            snprintf(opts.opwd_utf8, sizeof(opts.opwd_utf8), "%s", opwd ? opwd : "");
        }
        buf = fz_new_buffer(ctx, origlen ? origlen + 4096 : 4096);
        if (incremental) {
            if (!pdf_can_be_saved_incrementally(ctx, pdf))
                fz_throw(ctx, FZ_ERROR_GENERIC, "document cannot be saved incrementally");
            opts.do_incremental = 1;
            // The incremental update is a delta appended after the original
            // file content, so seed the output buffer with the original bytes.
            fz_append_data(ctx, buf, origdata, origlen);
        }
        out = fz_new_output_with_buffer(ctx, buf);
        pdf_write_document(ctx, pdf, out, &opts);
        fz_close_output(ctx, out);
        unsigned char *data = NULL;
        size_t n = fz_buffer_storage(ctx, buf, &data);
        result = (unsigned char *)malloc(n);
        if (result) { memcpy(result, data, n); *out_len = (int)n; }
    }
    fz_always(ctx) {
        fz_drop_output(ctx, out);
        fz_drop_buffer(ctx, buf);
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
	"os"
	"unsafe"
)

// Save options: fine-grained control over PDF serialization, mirroring the
// knobs PyMuPDF exposes on Document.save (garbage collection level, stream
// compression, linearization, pretty-printing, ASCII output, encryption, and
// incremental save).

// SaveOptions configures how a Document is serialized. The zero value writes an
// uncompressed, un-garbage-collected PDF; DefaultSaveOptions returns sensible
// production defaults. Encryption fields are honored only when Encrypt is true.
type SaveOptions struct {
	// Garbage is the dead-object collection level, 0–4:
	//  0 none, 1 collect unreferenced, 2 + compact xref, 3 + merge duplicates,
	//  4 + dedupe identical streams. Out-of-range values are clamped.
	Garbage int

	Deflate       bool // compress uncompressed streams (FlateDecode)
	DeflateImages bool // also recompress image streams
	DeflateFonts  bool // also recompress embedded font streams

	Clean  bool // sanitize and rewrite content streams
	Linear bool // linearize ("fast web view")
	ASCII  bool // emit only ASCII (escape binary)
	Pretty bool // pretty-print PDF objects

	Encrypt       bool   // serialize with AES-256 encryption
	UserPassword  string // required to open (when Encrypt)
	OwnerPassword string // grants full permissions (when Encrypt)
	Permissions   int    // permission bitmask; -1 grants all

	// Incremental appends changes to the original bytes instead of rewriting
	// the whole file. Valid only for documents opened from a file or stream;
	// incompatible with Garbage, Linear, and Encrypt.
	Incremental bool
}

// DefaultSaveOptions returns options suitable for most documents: garbage level
// 3 (merge duplicates) with stream deflation enabled.
func DefaultSaveOptions() SaveOptions {
	return SaveOptions{Garbage: 3, Deflate: true}
}

func bint(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// SaveBytesWithOptions serializes the document to PDF bytes using opts.
func (d *Document) SaveBytesWithOptions(opts SaveOptions) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	if opts.Incremental && (d.data == nil || d.dataLen == 0) {
		return nil, errors.New("gomupdf: incremental save requires a document opened from a file or stream")
	}
	cu := C.CString(opts.UserPassword)
	defer C.free(unsafe.Pointer(cu))
	co := C.CString(opts.OwnerPassword)
	defer C.free(unsafe.Pointer(co))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))

	var origPtr *C.uchar
	var origLen C.size_t
	if opts.Incremental {
		origPtr = (*C.uchar)(d.data)
		origLen = C.size_t(d.dataLen)
	}

	var outLen C.int
	ptr := C.gomupdf_save_opts(d.ctx, d.doc,
		C.int(opts.Garbage), bint(opts.Deflate), bint(opts.DeflateImages), bint(opts.DeflateFonts),
		bint(opts.Clean), bint(opts.Linear), bint(opts.ASCII), bint(opts.Pretty),
		bint(opts.Encrypt), cu, co, C.int(opts.Permissions),
		bint(opts.Incremental), origPtr, origLen,
		&outLen, errBuf, errBufLen)
	if ptr == nil {
		return nil, errors.New("gomupdf: save: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}

// SaveWithOptions writes the document to a PDF file using opts.
func (d *Document) SaveWithOptions(path string, opts SaveOptions) error {
	data, err := d.SaveBytesWithOptions(opts)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// EzSave writes the document to a file with DefaultSaveOptions — the
// convenience path equivalent to PyMuPDF's ez_save.
func (d *Document) EzSave(path string) error {
	return d.SaveWithOptions(path, DefaultSaveOptions())
}

// SaveIncremental appends pending changes to path using an incremental update.
// The document must have been opened from a file or stream. It is shorthand for
// SaveWithOptions with Incremental set.
func (d *Document) SaveIncremental(path string) error {
	return d.SaveWithOptions(path, SaveOptions{Incremental: true})
}
