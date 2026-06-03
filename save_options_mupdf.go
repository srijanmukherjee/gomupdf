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

// Serialize with AES-256 encryption using the given user/owner passwords.
static unsigned char *gomupdf_save_encrypted(fz_context *ctx, fz_document *doc,
                                             const char *user_pw, const char *owner_pw,
                                             int *out_len, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return NULL; }
    fz_buffer *buf = NULL;
    fz_output *out = NULL;
    unsigned char *result = NULL;
    fz_var(buf);
    fz_var(out);
    fz_try(ctx) {
        buf = fz_new_buffer(ctx, 4096);
        out = fz_new_output_with_buffer(ctx, buf);
        pdf_write_options opts;
        memset(&opts, 0, sizeof(opts));
        opts.do_garbage = 1;
        opts.do_encrypt = PDF_ENCRYPT_AES_256;
        opts.permissions = -1;
        snprintf(opts.upwd_utf8, sizeof(opts.upwd_utf8), "%s", user_pw ? user_pw : "");
        snprintf(opts.opwd_utf8, sizeof(opts.opwd_utf8), "%s", owner_pw ? owner_pw : "");
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
	"unsafe"
)

func cbint(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// save serializes with opts. The orig parameter is unused by the MuPDF backend:
// it retains the original bytes itself (d.data) for incremental saves.
func (d *mupdfDoc) save(opts SaveOptions, orig []byte) ([]byte, error) {
	_ = orig
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
		C.int(opts.Garbage), cbint(opts.Deflate), cbint(opts.DeflateImages), cbint(opts.DeflateFonts),
		cbint(opts.Clean), cbint(opts.Linear), cbint(opts.ASCII), cbint(opts.Pretty),
		cbint(opts.Encrypt), cu, co, C.int(opts.Permissions),
		cbint(opts.Incremental), origPtr, origLen,
		&outLen, errBuf, errBufLen)
	if ptr == nil {
		return nil, errors.New("gomupdf: save: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}

func (d *mupdfDoc) saveEncrypted(userPwd, ownerPwd string) ([]byte, error) {
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	cu := C.CString(userPwd)
	defer C.free(unsafe.Pointer(cu))
	co := C.CString(ownerPwd)
	defer C.free(unsafe.Pointer(co))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	var outLen C.int
	ptr := C.gomupdf_save_encrypted(d.ctx, d.doc, cu, co, &outLen, errBuf, errBufLen)
	if ptr == nil {
		return nil, errors.New("gomupdf: save encrypted: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}
