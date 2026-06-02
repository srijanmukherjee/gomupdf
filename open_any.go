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

// Create a context and open a document from a memory buffer using an explicit
// magic/filetype hint (e.g. "png", ".pdf", "application/epub+zip"). The buffer
// must outlive the doc. On success the new context is returned via *out_ctx and
// whether the document needs a password via *needs_pass.
static fz_document *gomupdf_open_any(const unsigned char *data, size_t len,
                                     const char *magic, fz_context **out_ctx,
                                     int *needs_pass, char *err, int errlen) {
    fz_context *ctx = fz_new_context(NULL, NULL, FZ_STORE_DEFAULT);
    if (!ctx) { snprintf(err, errlen, "failed to create context"); return NULL; }
    fz_document *doc = NULL;
    fz_stream *stream = NULL;
    fz_var(stream);
    fz_var(doc);
    fz_try(ctx) {
        fz_register_document_handlers(ctx);
        stream = fz_open_memory(ctx, data, len);
        doc = fz_open_document_with_stream(ctx, magic, stream);
        *needs_pass = fz_needs_password(ctx, doc);
    }
    fz_always(ctx) { fz_drop_stream(ctx, stream); }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        fz_drop_context(ctx);
        return NULL;
    }
    *out_ctx = ctx;
    return doc;
}

// Render every page of any source document into a new PDF and return the
// serialized PDF bytes (malloc'd + *out_len). Done entirely in the source
// context to avoid cross-context object access.
static unsigned char *gomupdf_convert_to_pdf(fz_context *ctx, fz_document *src,
                                             int *out_len, char *err, int errlen) {
    pdf_document *pdf = NULL;
    fz_buffer *outbuf = NULL;
    fz_output *out = NULL;
    unsigned char *result = NULL;
    fz_var(pdf);
    fz_var(outbuf);
    fz_var(out);
    fz_try(ctx) {
        pdf = pdf_create_document(ctx);
        int n = fz_count_pages(ctx, src);
        for (int i = 0; i < n; i++) {
            fz_page *page = NULL;
            fz_device *dev = NULL;
            fz_buffer *contents = NULL;
            pdf_obj *resources = NULL;
            pdf_obj *pageobj = NULL;
            fz_var(page);
            fz_var(dev);
            fz_var(contents);
            fz_var(resources);
            fz_var(pageobj);
            fz_try(ctx) {
                page = fz_load_page(ctx, src, i);
                fz_rect bounds = fz_bound_page(ctx, page);
                dev = pdf_page_write(ctx, pdf, bounds, &resources, &contents);
                fz_run_page(ctx, page, dev, fz_identity, NULL);
                fz_close_device(ctx, dev);
                pageobj = pdf_add_page(ctx, pdf, bounds, 0, resources, contents);
                pdf_insert_page(ctx, pdf, -1, pageobj);
            }
            fz_always(ctx) {
                pdf_drop_obj(ctx, pageobj);
                pdf_drop_obj(ctx, resources);
                fz_drop_buffer(ctx, contents);
                fz_drop_device(ctx, dev);
                fz_drop_page(ctx, page);
            }
            fz_catch(ctx) { fz_rethrow(ctx); }
        }
        outbuf = fz_new_buffer(ctx, 8192);
        out = fz_new_output_with_buffer(ctx, outbuf);
        pdf_write_options opts;
        memset(&opts, 0, sizeof(opts));
        opts.do_garbage = 3;
        opts.do_compress = 1;
        pdf_write_document(ctx, pdf, out, &opts);
        fz_close_output(ctx, out);
        unsigned char *data = NULL;
        size_t sz = fz_buffer_storage(ctx, outbuf, &data);
        result = (unsigned char *)malloc(sz);
        if (result) { memcpy(result, data, sz); *out_len = (int)sz; }
    }
    fz_always(ctx) {
        fz_drop_output(ctx, out);
        fz_drop_buffer(ctx, outbuf);
        fz_drop_document(ctx, (fz_document *)pdf);
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
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"
)

// Opening non-PDF formats. MuPDF's document handlers cover XPS/OXPS, EPUB,
// MOBI, FB2, CBZ, SVG, plain text, and raster images (PNG, JPEG, GIF, BMP,
// TIFF, …). These open read-only as ordinary Documents — text, geometry, and
// rendering all work — but only true PDFs can be written back; use ConvertToPDF
// to turn any opened document into a writable PDF Document.

// OpenAny opens a document of any MuPDF-supported format, inferring the format
// from the file extension. Use Open for the PDF-only fast path.
func OpenAny(filename string) (*Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	return OpenAnyStream(data, ext)
}

// OpenAnyStream opens in-memory bytes of any MuPDF-supported format. filetype
// is a format hint — a file extension ("png", "xps", "epub"), a leading-dot
// extension (".pdf"), or a MIME type. An empty filetype is treated as PDF.
func OpenAnyStream(data []byte, filetype string) (*Document, error) {
	if len(data) == 0 {
		return nil, errors.New("gomupdf: empty input")
	}
	if filetype == "" {
		filetype = ".pdf"
	}
	cdata := C.CBytes(data) // malloc'd copy; freed in Close
	cmagic := C.CString(filetype)
	defer C.free(unsafe.Pointer(cmagic))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))

	var ctx *C.fz_context
	var needsPass C.int
	doc := C.gomupdf_open_any((*C.uchar)(cdata), C.size_t(len(data)), cmagic, &ctx, &needsPass, errBuf, errBufLen)
	if doc == nil {
		C.free(cdata)
		return nil, errors.New("gomupdf: open failed: " + C.GoString(errBuf))
	}

	d := &Document{ctx: ctx, doc: doc, data: cdata, dataLen: len(data)}
	d.locked = needsPass != 0
	d.encrypted = d.locked
	runtime.SetFinalizer(d, (*Document).Close)
	return d, nil
}

// ConvertToPDF renders every page of the document into a freshly assembled PDF
// and returns it as a new, writable Document. The receiver is left untouched
// and must still be closed independently. For a document that is already PDF
// this produces a clean structural copy.
func (d *Document) ConvertToPDF() (*Document, error) {
	d.mu.Lock()
	if d.doc == nil {
		d.mu.Unlock()
		return nil, errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	var outLen C.int
	ptr := C.gomupdf_convert_to_pdf(d.ctx, d.doc, &outLen, errBuf, errBufLen)
	d.mu.Unlock()
	if ptr == nil {
		return nil, errors.New("gomupdf: convert to pdf: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(ptr))
	return OpenStream(C.GoBytes(unsafe.Pointer(ptr), outLen))
}
