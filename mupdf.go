// Package gomupdf is a focused cgo binding over the MuPDF C core
// (https://mupdf.com/core) for reading, extracting, rendering, and writing PDFs.
// It exposes a compact API: open a (possibly encrypted) PDF from a file or from
// memory, authenticate, and pull reading-order text and positioned geometry per
// page, along with rendering and document-assembly helpers.
//
// MuPDF + cgo discipline (the four rules):
//  1. fz_try/fz_catch use setjmp/longjmp. A longjmp must NEVER cross the cgo→Go
//     boundary, so every throwing fz_* call is wrapped inside a C helper that
//     does fz_try/fz_catch and returns an error string to Go.
//  2. fz_context is not thread-safe. Each Document owns its own context and is
//     guarded by a mutex; do not share a Document across goroutines without it.
//  3. All MuPDF objects are explicitly dropped (fz_drop_*); the C helpers clean
//     up in fz_always so a Go-side mistake cannot leak.
//  4. The in-memory PDF bytes backing fz_open_memory are NOT copied by MuPDF, so
//     we hold a C-allocated copy alive for the Document's whole lifetime.
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

// Open a PDF from a memory buffer. The buffer must outlive the returned doc.
// On error returns NULL and writes the MuPDF message into err (size errlen).
static fz_document *gomupdf_open_mem(fz_context *ctx, const unsigned char *data,
                                     size_t len, char *err, int errlen) {
    fz_document *doc = NULL;
    fz_stream *stream = NULL;
    fz_var(stream);
    fz_var(doc);
    fz_try(ctx) {
        stream = fz_open_memory(ctx, data, len);
        doc = fz_open_document_with_stream(ctx, ".pdf", stream);
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

// Extract reading-order plain text for one page (lines separated by '\n').
// Returns a malloc'd NUL-terminated string the caller must free, or NULL+err.
static char *gomupdf_page_text(fz_context *ctx, fz_document *doc, int pageno,
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
        buf = fz_new_buffer(ctx, 4096);
        out = fz_new_output_with_buffer(ctx, buf);
        fz_print_stext_page_as_text(ctx, out, stext);
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

// Char-level word extraction: walk the
// stext tree down to characters, accumulate into words breaking on a space or
// line end, and take each word's bbox as the union of its chars' boxes. Emits
// one word per line as "x0 y0 x1 y1 blockno lineno\t<text>". Returns malloc'd
// NUL-terminated string or NULL+err.
static char *gomupdf_page_words(fz_context *ctx, fz_document *doc, int pageno,
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
        buf = fz_new_buffer(ctx, 8192);
        out = fz_new_output_with_buffer(ctx, buf);

        int blockno = 0;
        for (fz_stext_block *block = stext->first_block; block; block = block->next) {
            if (block->type != FZ_STEXT_BLOCK_TEXT) { blockno++; continue; }
            int lineno = 0;
            for (fz_stext_line *line = block->u.t.first_line; line; line = line->next) {
                float x0 = 0, y0 = 0, x1 = 0, y1 = 0;
                int have = 0;
                fz_buffer *wbuf = fz_new_buffer(ctx, 64);
                fz_output *wout = fz_new_output_with_buffer(ctx, wbuf);
                // flush helper is inlined below twice (on space and at line end)
                for (fz_stext_char *ch = line->first_char; ch; ch = ch->next) {
                    int isspace = (ch->c == ' ' || ch->c == '\t' || ch->c == 0xA0);
                    if (isspace) {
                        if (have) {
                            fz_close_output(ctx, wout);
                            unsigned char *wd = NULL;
                            size_t wn = fz_buffer_storage(ctx, wbuf, &wd);
                            fz_write_printf(ctx, out, "%g %g %g %g %d %d\t", x0, y0, x1, y1, blockno, lineno);
                            fz_write_data(ctx, out, wd, wn);
                            fz_write_byte(ctx, out, '\n');
                            fz_drop_output(ctx, wout);
                            fz_drop_buffer(ctx, wbuf);
                            wbuf = fz_new_buffer(ctx, 64);
                            wout = fz_new_output_with_buffer(ctx, wbuf);
                            have = 0;
                        }
                        continue;
                    }
                    fz_rect r = fz_rect_from_quad(ch->quad);
                    if (!have) { x0 = r.x0; y0 = r.y0; x1 = r.x1; y1 = r.y1; have = 1; }
                    else {
                        if (r.x0 < x0) x0 = r.x0;
                        if (r.y0 < y0) y0 = r.y0;
                        if (r.x1 > x1) x1 = r.x1;
                        if (r.y1 > y1) y1 = r.y1;
                    }
                    fz_write_rune(ctx, wout, ch->c);
                }
                if (have) {
                    fz_close_output(ctx, wout);
                    unsigned char *wd = NULL;
                    size_t wn = fz_buffer_storage(ctx, wbuf, &wd);
                    fz_write_printf(ctx, out, "%g %g %g %g %d %d\t", x0, y0, x1, y1, blockno, lineno);
                    fz_write_data(ctx, out, wd, wn);
                    fz_write_byte(ctx, out, '\n');
                }
                fz_drop_output(ctx, wout);
                fz_drop_buffer(ctx, wbuf);
                lineno++;
            }
            blockno++;
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

// --- write / modify (the pdf_* mutation API) -------------------------------

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

// Serialize the document to PDF bytes. garbage>0 enables garbage collection of
// unused objects. Returns malloc'd blob + *out_len, or NULL+err.
static unsigned char *gomupdf_save(fz_context *ctx, fz_document *doc, int garbage,
                                   int *out_len, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) {
        snprintf(err, errlen, "not a PDF document");
        return NULL;
    }
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
        if (garbage) opts.do_garbage = 1;
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

// Append a blank page of the given size at the end.
static int gomupdf_add_blank_page(fz_context *ctx, fz_document *doc, float w, float h,
                                  char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_buffer *contents = NULL;
    pdf_obj *resources = NULL;
    pdf_obj *page = NULL;
    fz_var(contents);
    fz_var(resources);
    fz_var(page);
    fz_try(ctx) {
        fz_rect mediabox = fz_make_rect(0, 0, w, h);
        contents = fz_new_buffer(ctx, 16);
        resources = pdf_new_dict(ctx, pdf, 1);
        page = pdf_add_page(ctx, pdf, mediabox, 0, resources, contents);
        pdf_insert_page(ctx, pdf, pdf_count_pages(ctx, pdf), page);
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, page);
        pdf_drop_obj(ctx, resources);
        fz_drop_buffer(ctx, contents);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

static int gomupdf_delete_page(fz_context *ctx, fz_document *doc, int n, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) { pdf_delete_page(ctx, pdf, n); }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Append all pages of a source PDF (given as bytes, opened in the destination's
// own context to avoid cross-context object access) to the end of dst.
static int gomupdf_graft_bytes(fz_context *ctx, fz_document *dst_doc,
                               const unsigned char *srcdata, size_t srclen,
                               const char *password, char *err, int errlen) {
    pdf_document *dst = pdf_specifics(ctx, dst_doc);
    if (!dst) { snprintf(err, errlen, "destination is not a PDF"); return -1; }
    fz_stream *stream = NULL;
    fz_document *srcfz = NULL;
    pdf_graft_map *map = NULL;
    fz_var(stream);
    fz_var(srcfz);
    fz_var(map);
    fz_try(ctx) {
        stream = fz_open_memory(ctx, srcdata, srclen);
        srcfz = fz_open_document_with_stream(ctx, ".pdf", stream);
        if (password && password[0] && fz_needs_password(ctx, srcfz))
            fz_authenticate_password(ctx, srcfz, password);
        pdf_document *src = pdf_specifics(ctx, srcfz);
        if (!src) fz_throw(ctx, FZ_ERROR_GENERIC, "source is not a PDF");
        int n = pdf_count_pages(ctx, src);
        map = pdf_new_graft_map(ctx, dst);
        for (int i = 0; i < n; i++)
            pdf_graft_mapped_page(ctx, map, -1, src, i);
    }
    fz_always(ctx) {
        pdf_drop_graft_map(ctx, map);
        fz_drop_document(ctx, srcfz);
        fz_drop_stream(ctx, stream);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Insert a line of Helvetica text at (x,y) on a page (origin = baseline).
static int gomupdf_insert_text(fz_context *ctx, fz_document *doc, int pageno,
                               float x, float y, float size, const char *text,
                               char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_buffer *cbuf = NULL;
    pdf_obj *stream = NULL;
    fz_var(page);
    fz_var(cbuf);
    fz_var(stream);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_obj *pobj = page->obj;
        // ensure /Resources /Font /F0 = Helvetica
        pdf_obj *res = pdf_dict_get(ctx, pobj, PDF_NAME(Resources));
        if (!res) {
            res = pdf_dict_put_dict(ctx, pobj, PDF_NAME(Resources), 2);
        }
        pdf_obj *fonts = pdf_dict_get(ctx, res, PDF_NAME(Font));
        if (!fonts) {
            fonts = pdf_dict_put_dict(ctx, res, PDF_NAME(Font), 1);
        }
        pdf_obj *font = pdf_new_dict(ctx, pdf, 4);
        pdf_dict_put(ctx, font, PDF_NAME(Type), PDF_NAME(Font));
        pdf_dict_put(ctx, font, PDF_NAME(Subtype), PDF_NAME(Type1));
        pdf_dict_put_drop(ctx, font, PDF_NAME(BaseFont), pdf_new_name(ctx, "Helvetica"));
        pdf_dict_puts_drop(ctx, fonts, "F0", pdf_add_object_drop(ctx, pdf, font));

        // build content: BT /F0 size Tf x y Td (text) Tj ET
        cbuf = fz_new_buffer(ctx, 128);
        fz_append_printf(ctx, cbuf, "\nq BT /F0 %g Tf %g %g Td (", size, x, y);
        for (const char *c = text; *c; c++) {
            if (*c == '(' || *c == ')' || *c == '\\') fz_append_byte(ctx, cbuf, '\\');
            fz_append_byte(ctx, cbuf, *c);
        }
        fz_append_printf(ctx, cbuf, ") Tj ET Q\n");

        // append as an additional content stream
        stream = pdf_add_stream(ctx, pdf, cbuf, NULL, 0);
        pdf_obj *contents = pdf_dict_get(ctx, pobj, PDF_NAME(Contents));
        if (pdf_is_array(ctx, contents)) {
            pdf_array_push(ctx, contents, stream);
        } else {
            pdf_obj *arr = pdf_new_array(ctx, pdf, 2);
            if (contents) pdf_array_push(ctx, arr, contents);
            pdf_array_push(ctx, arr, stream);
            pdf_dict_put_drop(ctx, pobj, PDF_NAME(Contents), arr);
        }
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, stream);
        fz_drop_buffer(ctx, cbuf);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Insert an image (given as encoded bytes) into the rect (x,y,w,h) on a page.
static int gomupdf_insert_image(fz_context *ctx, fz_document *doc, int pageno,
                                float x, float y, float w, float h,
                                const unsigned char *imgdata, size_t imglen,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_buffer *ibuf = NULL;
    fz_image *img = NULL;
    pdf_obj *imgref = NULL;
    fz_buffer *cbuf = NULL;
    pdf_obj *stream = NULL;
    fz_var(page);
    fz_var(ibuf);
    fz_var(img);
    fz_var(imgref);
    fz_var(cbuf);
    fz_var(stream);
    fz_try(ctx) {
        ibuf = fz_new_buffer_from_copied_data(ctx, imgdata, imglen);
        img = fz_new_image_from_buffer(ctx, ibuf);
        imgref = pdf_add_image(ctx, pdf, img);

        page = pdf_load_page(ctx, pdf, pageno);
        pdf_obj *pobj = page->obj;
        pdf_obj *res = pdf_dict_get(ctx, pobj, PDF_NAME(Resources));
        if (!res) res = pdf_dict_put_dict(ctx, pobj, PDF_NAME(Resources), 2);
        pdf_obj *xobj = pdf_dict_get(ctx, res, PDF_NAME(XObject));
        if (!xobj) xobj = pdf_dict_put_dict(ctx, res, PDF_NAME(XObject), 1);
        char name[32];
        snprintf(name, sizeof(name), "GmImg%d", pdf_dict_len(ctx, xobj));
        pdf_dict_puts(ctx, xobj, name, imgref);

        cbuf = fz_new_buffer(ctx, 128);
        fz_append_printf(ctx, cbuf, "\nq %g 0 0 %g %g %g cm /%s Do Q\n", w, h, x, y, name);
        stream = pdf_add_stream(ctx, pdf, cbuf, NULL, 0);
        pdf_obj *contents = pdf_dict_get(ctx, pobj, PDF_NAME(Contents));
        if (pdf_is_array(ctx, contents)) {
            pdf_array_push(ctx, contents, stream);
        } else {
            pdf_obj *arr = pdf_new_array(ctx, pdf, 2);
            if (contents) pdf_array_push(ctx, arr, contents);
            pdf_array_push(ctx, arr, stream);
            pdf_dict_put_drop(ctx, pobj, PDF_NAME(Contents), arr);
        }
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, stream);
        fz_drop_buffer(ctx, cbuf);
        pdf_drop_obj(ctx, imgref);
        fz_drop_image(ctx, img);
        fz_drop_buffer(ctx, ibuf);
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Add a rectangle annotation on a page.
static int gomupdf_add_rect_annot(fz_context *ctx, fz_document *doc, int pageno,
                                  float x0, float y0, float x1, float y1,
                                  char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    pdf_page *page = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = pdf_load_page(ctx, pdf, pageno);
        pdf_annot *annot = pdf_create_annot(ctx, page, PDF_ANNOT_SQUARE);
        fz_rect r = fz_make_rect(x0, y0, x1, y1);
        pdf_set_annot_rect(ctx, annot, r);
        pdf_update_annot(ctx, annot);
        pdf_drop_annot(ctx, annot);
    }
    fz_always(ctx) {
        fz_drop_page(ctx, (fz_page *)page);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Render a page to a raster. Returns a malloc'd blob:
// 5 int32 header [w, h, n, stride, alpha] followed by the raw samples. *out_len
// is set to the total byte length. Returns NULL+err on failure.
static unsigned char *gomupdf_pixmap(fz_context *ctx, fz_document *doc, int pageno,
                                     float zoom, int gray, int alpha,
                                     int *out_len, char *err, int errlen) {
    fz_pixmap *pix = NULL;
    unsigned char *result = NULL;
    fz_var(pix);
    fz_try(ctx) {
        fz_matrix m = fz_scale(zoom, zoom);
        fz_colorspace *cs = gray ? fz_device_gray(ctx) : fz_device_rgb(ctx);
        pix = fz_new_pixmap_from_page_number(ctx, doc, pageno, m, cs, alpha);
        int32_t hdr[5];
        hdr[0] = pix->w;
        hdr[1] = pix->h;
        hdr[2] = pix->n;
        hdr[3] = pix->stride;
        hdr[4] = pix->alpha;
        size_t samples = (size_t)pix->stride * pix->h;
        size_t total = sizeof(hdr) + samples;
        result = (unsigned char *)malloc(total);
        if (result) {
            memcpy(result, hdr, sizeof(hdr));
            memcpy(result + sizeof(hdr), pix->samples, samples);
            *out_len = (int)total;
        }
    }
    fz_always(ctx) {
        fz_drop_pixmap(ctx, pix);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

// --- images (get_images / get_image_bbox / extract_image) ------------------
// A trace device captures fill_image calls: the placement bbox comes from the
// image's unit rect transformed by the ctm, dimensions/bpc/colorspace from the
// fz_image, and the encoding type from its compressed buffer.

static const char *gomupdf_image_ext(fz_context *ctx, fz_image *img) {
    fz_compressed_buffer *cb = fz_compressed_image_buffer(ctx, img);
    if (!cb) return "png"; // uncompressed → we re-encode as PNG
    switch (cb->params.type) {
    case FZ_IMAGE_JPEG: return "jpeg";
    case FZ_IMAGE_JPX:  return "jpx";
    case FZ_IMAGE_PNG:  return "png";
    case FZ_IMAGE_JBIG2:return "jb2";
    case FZ_IMAGE_FAX:  return "fax";
    case FZ_IMAGE_TIFF: return "tiff";
    case FZ_IMAGE_BMP:  return "bmp";
    case FZ_IMAGE_GIF:  return "gif";
    default:            return "png";
    }
}

typedef struct {
    fz_device super;
    fz_buffer *buf;
} gomupdf_img_meta_device;

static void gomupdf_img_meta_fill(fz_context *ctx, fz_device *dev, fz_image *img,
                                  fz_matrix ctm, float alpha, fz_color_params cp) {
    fz_buffer *buf = ((gomupdf_img_meta_device *)dev)->buf;
    fz_rect r = fz_transform_rect(fz_unit_rect, ctm);
    int n = img->colorspace ? fz_colorspace_n(ctx, img->colorspace) : 0;
    fz_append_printf(ctx, buf, "IMG %g %g %g %g %d %d %d %d %s\n",
                     r.x0, r.y0, r.x1, r.y1, img->w, img->h, img->bpc, n,
                     gomupdf_image_ext(ctx, img));
}

static char *gomupdf_images(fz_context *ctx, fz_document *doc, int pageno,
                            char *err, int errlen) {
    fz_page *page = NULL;
    fz_device *dev = NULL;
    fz_buffer *buf = NULL;
    char *result = NULL;
    fz_var(page);
    fz_var(dev);
    fz_var(buf);
    fz_try(ctx) {
        page = fz_load_page(ctx, doc, pageno);
        buf = fz_new_buffer(ctx, 1024);
        dev = fz_new_device_of_size(ctx, sizeof(gomupdf_img_meta_device));
        dev->fill_image = gomupdf_img_meta_fill;
        ((gomupdf_img_meta_device *)dev)->buf = buf;
        fz_run_page(ctx, page, dev, fz_identity, NULL);
        fz_close_device(ctx, dev);
        unsigned char *data = NULL;
        size_t sz = fz_buffer_storage(ctx, buf, &data);
        result = (char *)malloc(sz + 1);
        if (result) { memcpy(result, data, sz); result[sz] = '\0'; }
    }
    fz_always(ctx) {
        fz_drop_device(ctx, dev);
        fz_drop_buffer(ctx, buf);
        fz_drop_page(ctx, page);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

// Extract the bytes of the image at the given fill-order index. Returns the
// original encoded bytes when available (jpeg/jpx/png/…), else a PNG re-encode.
typedef struct {
    fz_device super;
    fz_context *ctx;
    int target;
    int count;
    unsigned char *result;
    int len;
    char ext[8];
} gomupdf_img_extract_device;

static void gomupdf_img_extract_fill(fz_context *ctx, fz_device *dev, fz_image *img,
                                     fz_matrix ctm, float alpha, fz_color_params cp) {
    gomupdf_img_extract_device *d = (gomupdf_img_extract_device *)dev;
    if (d->count == d->target && d->result == NULL) {
        fz_compressed_buffer *cb = fz_compressed_image_buffer(ctx, img);
        snprintf(d->ext, sizeof(d->ext), "%s", gomupdf_image_ext(ctx, img));
        if (cb && cb->buffer) {
            unsigned char *data = NULL;
            size_t n = fz_buffer_storage(ctx, cb->buffer, &data);
            d->result = (unsigned char *)malloc(n);
            if (d->result) { memcpy(d->result, data, n); d->len = (int)n; }
        } else {
            fz_pixmap *pix = NULL;
            fz_buffer *png = NULL;
            fz_var(pix);
            fz_var(png);
            fz_try(ctx) {
                pix = fz_get_pixmap_from_image(ctx, img, NULL, NULL, NULL, NULL);
                png = fz_new_buffer_from_pixmap_as_png(ctx, pix, fz_default_color_params);
                unsigned char *data = NULL;
                size_t n = fz_buffer_storage(ctx, png, &data);
                d->result = (unsigned char *)malloc(n);
                if (d->result) { memcpy(d->result, data, n); d->len = (int)n; }
                snprintf(d->ext, sizeof(d->ext), "png");
            }
            fz_always(ctx) {
                fz_drop_buffer(ctx, png);
                fz_drop_pixmap(ctx, pix);
            }
            fz_catch(ctx) { d->result = NULL; }
        }
    }
    d->count++;
}

static unsigned char *gomupdf_image_bytes(fz_context *ctx, fz_document *doc, int pageno,
                                          int index, int *out_len, char *ext_out,
                                          char *err, int errlen) {
    fz_page *page = NULL;
    gomupdf_img_extract_device *dev = NULL;
    unsigned char *result = NULL;
    fz_var(page);
    fz_var(dev);
    fz_try(ctx) {
        page = fz_load_page(ctx, doc, pageno);
        dev = (gomupdf_img_extract_device *)fz_new_device_of_size(ctx, sizeof(gomupdf_img_extract_device));
        dev->super.fill_image = gomupdf_img_extract_fill;
        dev->ctx = ctx;
        dev->target = index;
        dev->count = 0;
        dev->result = NULL;
        dev->len = 0;
        dev->ext[0] = '\0';
        fz_run_page(ctx, page, (fz_device *)dev, fz_identity, NULL);
        fz_close_device(ctx, (fz_device *)dev);
        result = dev->result;
        *out_len = dev->len;
        snprintf(ext_out, 8, "%s", dev->ext);
    }
    fz_always(ctx) {
        fz_drop_device(ctx, (fz_device *)dev);
        fz_drop_page(ctx, page);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

// --- drawings (get_drawings) via a path-capturing trace device -------------
// A custom fz_device records every fill/stroke path. fz_walk_path emits the
// path's segments (transformed by the current matrix); each path is prefixed by
// "P <type> <r> <g> <b> <width> <linejoin> <alpha>".

typedef struct {
    fz_context *ctx;
    fz_buffer *buf;
    fz_matrix ctm;
    float sx, sy;   // current subpath start (transformed)
    int have_start;
} gomupdf_pw_arg;

static void gomupdf_pw_moveto(fz_context *ctx, void *arg, float x, float y) {
    gomupdf_pw_arg *a = (gomupdf_pw_arg *)arg;
    fz_point p = fz_transform_point_xy(x, y, a->ctm);
    a->sx = p.x; a->sy = p.y; a->have_start = 1;
    fz_append_printf(ctx, a->buf, "m %g %g\n", p.x, p.y);
}
static void gomupdf_pw_lineto(fz_context *ctx, void *arg, float x, float y) {
    gomupdf_pw_arg *a = (gomupdf_pw_arg *)arg;
    fz_point p = fz_transform_point_xy(x, y, a->ctm);
    fz_append_printf(ctx, a->buf, "l %g %g\n", p.x, p.y);
}
static void gomupdf_pw_curveto(fz_context *ctx, void *arg, float x1, float y1,
                               float x2, float y2, float x3, float y3) {
    gomupdf_pw_arg *a = (gomupdf_pw_arg *)arg;
    fz_point p1 = fz_transform_point_xy(x1, y1, a->ctm);
    fz_point p2 = fz_transform_point_xy(x2, y2, a->ctm);
    fz_point p3 = fz_transform_point_xy(x3, y3, a->ctm);
    fz_append_printf(ctx, a->buf, "c %g %g %g %g %g %g\n", p1.x, p1.y, p2.x, p2.y, p3.x, p3.y);
}
static void gomupdf_pw_closepath(fz_context *ctx, void *arg) {
    gomupdf_pw_arg *a = (gomupdf_pw_arg *)arg;
    // Emit the implicit closing segment back to the subpath start so the 4th
    // side of a rectangle (and any closed polygon) is captured as a line.
    if (a->have_start)
        fz_append_printf(ctx, a->buf, "l %g %g\n", a->sx, a->sy);
    fz_append_printf(ctx, a->buf, "h\n");
}
static const fz_path_walker gomupdf_path_walker = {
    .moveto = gomupdf_pw_moveto,
    .lineto = gomupdf_pw_lineto,
    .curveto = gomupdf_pw_curveto,
    .closepath = gomupdf_pw_closepath,
};

typedef struct {
    fz_device super;
    fz_buffer *buf;
} gomupdf_draw_device;

static void gomupdf_emit_rgb(fz_context *ctx, fz_buffer *buf, fz_colorspace *cs,
                             const float *color, fz_color_params cp) {
    float rgb[3] = {0, 0, 0};
    if (cs && color)
        fz_convert_color(ctx, cs, color, fz_device_rgb(ctx), rgb, NULL, cp);
    fz_append_printf(ctx, buf, "%g %g %g", rgb[0], rgb[1], rgb[2]);
}

static void gomupdf_walk(fz_context *ctx, fz_buffer *buf, const fz_path *path, fz_matrix ctm) {
    gomupdf_pw_arg a = {ctx, buf, ctm};
    fz_walk_path(ctx, path, &gomupdf_path_walker, &a);
}

static void gomupdf_fill_path(fz_context *ctx, fz_device *dev, const fz_path *path,
                              int even_odd, fz_matrix ctm, fz_colorspace *cs,
                              const float *color, float alpha, fz_color_params cp) {
    fz_buffer *buf = ((gomupdf_draw_device *)dev)->buf;
    fz_append_printf(ctx, buf, "P f ");
    gomupdf_emit_rgb(ctx, buf, cs, color, cp);
    fz_append_printf(ctx, buf, " 0 0 %g\n", alpha);
    gomupdf_walk(ctx, buf, path, ctm);
}

static void gomupdf_stroke_path(fz_context *ctx, fz_device *dev, const fz_path *path,
                                const fz_stroke_state *st, fz_matrix ctm, fz_colorspace *cs,
                                const float *color, float alpha, fz_color_params cp) {
    fz_buffer *buf = ((gomupdf_draw_device *)dev)->buf;
    float expn = fz_matrix_expansion(ctm);
    float w = (st ? st->linewidth : 1.0f) * expn;
    int lj = st ? (int)st->linejoin : 0;
    fz_append_printf(ctx, buf, "P s ");
    gomupdf_emit_rgb(ctx, buf, cs, color, cp);
    fz_append_printf(ctx, buf, " %g %d %g\n", w, lj, alpha);
    gomupdf_walk(ctx, buf, path, ctm);
}

static char *gomupdf_drawings(fz_context *ctx, fz_document *doc, int pageno,
                              char *err, int errlen) {
    fz_page *page = NULL;
    fz_device *dev = NULL;
    fz_buffer *buf = NULL;
    char *result = NULL;
    fz_var(page);
    fz_var(dev);
    fz_var(buf);
    fz_try(ctx) {
        page = fz_load_page(ctx, doc, pageno);
        buf = fz_new_buffer(ctx, 4096);
        dev = fz_new_device_of_size(ctx, sizeof(gomupdf_draw_device));
        dev->fill_path = gomupdf_fill_path;
        dev->stroke_path = gomupdf_stroke_path;
        ((gomupdf_draw_device *)dev)->buf = buf;
        fz_run_page(ctx, page, dev, fz_identity, NULL);
        fz_close_device(ctx, dev);
        unsigned char *data = NULL;
        size_t sz = fz_buffer_storage(ctx, buf, &data);
        result = (char *)malloc(sz + 1);
        if (result) { memcpy(result, data, sz); result[sz] = '\0'; }
    }
    fz_always(ctx) {
        fz_drop_device(ctx, dev);
        fz_drop_buffer(ctx, buf);
        fz_drop_page(ctx, page);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

// Page bounding box. Emits "x0 y0 x1 y1".
static char *gomupdf_page_bound(fz_context *ctx, fz_document *doc, int pageno,
                                char *err, int errlen) {
    fz_page *page = NULL;
    char *result = NULL;
    fz_var(page);
    fz_try(ctx) {
        page = fz_load_page(ctx, doc, pageno);
        fz_rect r = fz_bound_page(ctx, page);
        result = (char *)malloc(128);
        if (result)
            snprintf(result, 128, "%g %g %g %g", r.x0, r.y0, r.x1, r.y1);
    }
    fz_always(ctx) {
        fz_drop_page(ctx, page);
    }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return NULL;
    }
    return result;
}

// Walk the outline tree emitting "level\tpage\ttitle" per node (depth-first).
static void gomupdf_emit_outline(fz_context *ctx, fz_document *doc, fz_output *out,
                                 fz_outline *node, int level) {
    for (; node; node = node->next) {
        int pno = fz_page_number_from_location(ctx, doc, node->page);
        fz_write_printf(ctx, out, "%d\t%d\t", level, pno);
        if (node->title) {
            for (const char *c = node->title; *c; c++)
                fz_write_byte(ctx, out, (*c == '\n' || *c == '\t') ? ' ' : *c);
        }
        fz_write_byte(ctx, out, '\n');
        if (node->down)
            gomupdf_emit_outline(ctx, doc, out, node->down, level + 1);
    }
}

// Table of contents. Returns malloc'd string or NULL+err.
static char *gomupdf_toc(fz_context *ctx, fz_document *doc, char *err, int errlen) {
    fz_outline *ol = NULL;
    fz_buffer *buf = NULL;
    fz_output *out = NULL;
    char *result = NULL;
    fz_var(ol);
    fz_var(buf);
    fz_var(out);
    fz_try(ctx) {
        ol = fz_load_outline(ctx, doc);
        buf = fz_new_buffer(ctx, 1024);
        out = fz_new_output_with_buffer(ctx, buf);
        if (ol)
            gomupdf_emit_outline(ctx, doc, out, ol, 1);
        fz_close_output(ctx, out);
        unsigned char *data = NULL;
        size_t sz = fz_buffer_storage(ctx, buf, &data);
        result = (char *)malloc(sz + 1);
        if (result) { memcpy(result, data, sz); result[sz] = '\0'; }
    }
    fz_always(ctx) {
        fz_drop_outline(ctx, ol);
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

// Page links. Emits "x0 y0 x1 y1\turi" per link.
static char *gomupdf_links(fz_context *ctx, fz_document *doc, int pageno,
                           char *err, int errlen) {
    fz_page *page = NULL;
    fz_link *links = NULL;
    fz_buffer *buf = NULL;
    fz_output *out = NULL;
    char *result = NULL;
    fz_var(page);
    fz_var(links);
    fz_var(buf);
    fz_var(out);
    fz_try(ctx) {
        page = fz_load_page(ctx, doc, pageno);
        links = fz_load_links(ctx, page);
        buf = fz_new_buffer(ctx, 1024);
        out = fz_new_output_with_buffer(ctx, buf);
        for (fz_link *l = links; l; l = l->next) {
            fz_write_printf(ctx, out, "%g %g %g %g\t", l->rect.x0, l->rect.y0, l->rect.x1, l->rect.y1);
            if (l->uri) {
                for (const char *c = l->uri; *c; c++)
                    fz_write_byte(ctx, out, (*c == '\n' || *c == '\t') ? ' ' : *c);
            }
            fz_write_byte(ctx, out, '\n');
        }
        fz_close_output(ctx, out);
        unsigned char *data = NULL;
        size_t sz = fz_buffer_storage(ctx, buf, &data);
        result = (char *)malloc(sz + 1);
        if (result) { memcpy(result, data, sz); result[sz] = '\0'; }
    }
    fz_always(ctx) {
        fz_drop_link(ctx, links);
        fz_drop_page(ctx, page);
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

// Search for needle on a page. Emits one hit per line as
// 8 floats "ul.x ul.y ur.x ur.y ll.x ll.y lr.x lr.y" (the hit quad). Returns
// malloc'd NUL-terminated string or NULL+err.
static char *gomupdf_search(fz_context *ctx, fz_document *doc, int pageno,
                            const char *needle, char *err, int errlen) {
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
        enum { MAXHITS = 512 };
        fz_quad quads[MAXHITS];
        int n = fz_search_stext_page(ctx, stext, needle, NULL, quads, MAXHITS);
        buf = fz_new_buffer(ctx, 1024);
        out = fz_new_output_with_buffer(ctx, buf);
        for (int i = 0; i < n; i++) {
            fz_quad q = quads[i];
            fz_write_printf(ctx, out, "%g %g %g %g %g %g %g %g\n",
                            q.ul.x, q.ul.y, q.ur.x, q.ur.y, q.ll.x, q.ll.y, q.lr.x, q.lr.y);
        }
        fz_close_output(ctx, out);
        unsigned char *data = NULL;
        size_t sz = fz_buffer_storage(ctx, buf, &data);
        result = (char *)malloc(sz + 1);
        if (result) { memcpy(result, data, sz); result[sz] = '\0'; }
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

// Look up a metadata value by key (e.g. "info:Title", "format"). Returns the
// value length (>=0) or -1 if absent; writes into buf.
static int gomupdf_lookup_meta(fz_context *ctx, fz_document *doc, const char *key,
                               char *buf, int size) {
    int n = -1;
    fz_try(ctx) { n = fz_lookup_metadata(ctx, doc, key, buf, size); }
    fz_catch(ctx) { return -1; }
    return n;
}

// Structured text as JSON (blocks → lines → chars, each with bbox).
// Returns malloc'd NUL-terminated string or NULL+err.
static char *gomupdf_page_stext_json(fz_context *ctx, fz_document *doc, int pageno,
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
        buf = fz_new_buffer(ctx, 8192);
        out = fz_new_output_with_buffer(ctx, buf);
        fz_print_stext_page_as_json(ctx, out, stext, 1.0f);
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
	"iter"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

const errBufLen = 512

// ---------------------------------------------------------------------------
// Document and page API.
//
// A Document is opened from a file (Open) or from in-memory bytes (OpenStream),
// optionally unlocked with Authenticate, and queried via NeedsPass, PageCount,
// LoadPage, and Pages. Each Page exposes text and geometry extraction. Always
// Close a Document when done.
// ---------------------------------------------------------------------------

// Document is an open PDF. Methods are serialized internally; a Document binds
// one MuPDF context. Always Close it.
type Document struct {
	mu        sync.Mutex
	ctx       *C.fz_context
	doc       *C.fz_document
	data      unsafe.Pointer // C-malloc'd copy backing the in-memory stream
	dataLen   int            // length of the original bytes at data (for incremental save)
	locked    bool           // encrypted and not yet authenticated
	encrypted bool           // whether the PDF was password-protected at open
}

// Open opens a PDF document from a file path.
func Open(filename string) (*Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return OpenStream(data)
}

// OpenStream opens a PDF from in-memory bytes. It does
// not fail on encryption; the returned Document is left locked (NeedsPass()
// true) until Authenticate succeeds.
func OpenStream(pdf []byte) (*Document, error) {
	if len(pdf) == 0 {
		return nil, errors.New("gomupdf: empty input")
	}
	ctx := C.gomupdf_new_context()
	if ctx == nil {
		return nil, errors.New("gomupdf: failed to create context")
	}
	cdata := C.CBytes(pdf) // malloc'd copy; freed in Close
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))

	doc := C.gomupdf_open_mem(ctx, (*C.uchar)(cdata), C.size_t(len(pdf)), errBuf, errBufLen)
	if doc == nil {
		C.free(cdata)
		C.fz_drop_context(ctx)
		return nil, errors.New("gomupdf: open failed: " + C.GoString(errBuf))
	}

	d := &Document{ctx: ctx, doc: doc, data: cdata, dataLen: len(pdf)}
	d.locked = C.gomupdf_needs_password(ctx, doc) != 0
	d.encrypted = d.locked
	runtime.SetFinalizer(d, (*Document).Close)
	return d, nil
}

// OpenWithPassword opens a file and authenticates in one step, returning an
// error if the password is wrong (QoL over Open + Authenticate).
func OpenWithPassword(filename, password string) (*Document, error) {
	d, err := Open(filename)
	if err != nil {
		return nil, err
	}
	if d.NeedsPass() && !d.Authenticate(password) {
		d.Close()
		return nil, errors.New("gomupdf: incorrect password")
	}
	return d, nil
}

// OpenStreamWithPassword is OpenWithPassword for in-memory bytes.
func OpenStreamWithPassword(pdf []byte, password string) (*Document, error) {
	d, err := OpenStream(pdf)
	if err != nil {
		return nil, err
	}
	if d.NeedsPass() && !d.Authenticate(password) {
		d.Close()
		return nil, errors.New("gomupdf: incorrect password")
	}
	return d, nil
}

// IsEncrypted reports whether the PDF was password-protected (stays true even
// after successful authentication, unlike NeedsPass).
func (d *Document) IsEncrypted() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.encrypted
}

// NeedsPass reports whether the document is encrypted and not yet unlocked.
// Unlike MuPDF's static flag, this flips to false once Authenticate succeeds.
func (d *Document) NeedsPass() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.locked
}

// Authenticate tries a password and returns true if the document is now
// readable (correct password, or it was never locked).
func (d *Document) Authenticate(password string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return false
	}
	cpw := C.CString(password)
	defer C.free(unsafe.Pointer(cpw))
	ok := C.gomupdf_authenticate(d.ctx, d.doc, cpw) != 0
	if ok {
		d.locked = false
	}
	return ok
}

// PageCount returns the number of pages. Returns 0 if the
// document is closed or unreadable.
func (d *Document) PageCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
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

// LoadPage returns a handle to page i (0-based).
// The page is lightweight (it defers to the document on each call); no separate
// Close is required.
func (d *Document) LoadPage(i int) (*Page, error) {
	if i < 0 || i >= d.PageCount() {
		return nil, errors.New("gomupdf: page out of range")
	}
	return &Page{doc: d, Number: i}, nil
}

// Pages iterates pages in order. Use as:
//
//	for i, page := range doc.Pages() { ... }
func (d *Document) Pages() iter.Seq2[int, *Page] {
	return func(yield func(int, *Page) bool) {
		n := d.PageCount()
		for i := 0; i < n; i++ {
			if !yield(i, &Page{doc: d, Number: i}) {
				return
			}
		}
	}
}

// rawText pulls page i's reading-order text (lines '\n'-joined).
func (d *Document) rawText(i int) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_text(d.ctx, d.doc, C.int(i), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: page text: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

// Close releases all native resources. Safe to call twice.
func (d *Document) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
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
	runtime.SetFinalizer(d, nil)
}

// Page is a handle to a single page.
type Page struct {
	doc    *Document
	Number int // 0-based page index
}

// GetText returns the page's reading-order plain text.
func (p *Page) GetText() (string, error) {
	return p.doc.rawText(p.Number)
}

// Bound returns the page's bounding box in points.
func (p *Page) Bound() (geometry.Rect, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return geometry.Rect{}, errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_bound(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return geometry.Rect{}, errors.New("gomupdf: bound: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	f := strings.Fields(C.GoString(cstr))
	if len(f) != 4 {
		return geometry.Rect{}, errors.New("gomupdf: bad bound output")
	}
	x0, _ := strconv.ParseFloat(f[0], 64)
	y0, _ := strconv.ParseFloat(f[1], 64)
	x1, _ := strconv.ParseFloat(f[2], 64)
	y1, _ := strconv.ParseFloat(f[3], 64)
	return geometry.Rect{X0: x0, Y0: y0, X1: x1, Y1: y1}, nil
}

// --- write API (Go entry points; cgo lives here) ---------------------------

// NewPDF creates a new, empty PDF document.
func NewPDF() (*Document, error) {
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
	d := &Document{ctx: ctx, doc: doc}
	runtime.SetFinalizer(d, (*Document).Close)
	return d, nil
}

// SaveBytes serializes the document to PDF bytes. garbage collects unused objects.
func (d *Document) SaveBytes(garbage bool) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	g := C.int(0)
	if garbage {
		g = 1
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	var outLen C.int
	ptr := C.gomupdf_save(d.ctx, d.doc, g, &outLen, errBuf, errBufLen)
	if ptr == nil {
		return nil, errors.New("gomupdf: save: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}

// Save writes the document to a PDF file.
func (d *Document) Save(path string, garbage bool) error {
	data, err := d.SaveBytes(garbage)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// SaveEncryptedBytes serializes the document with AES-256 encryption. userPwd
// is required to open; ownerPwd grants full permissions (may be "").
func (d *Document) SaveEncryptedBytes(userPwd, ownerPwd string) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
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

// NewPage appends a blank page of the given size (points).
func (d *Document) NewPage(width, height float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_blank_page(d.ctx, d.doc, C.float(width), C.float(height), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: new page: " + C.GoString(errBuf))
	}
	return nil
}

// DeletePage removes page n (0-based).
func (d *Document) DeletePage(n int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_delete_page(d.ctx, d.doc, C.int(n), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: delete page: " + C.GoString(errBuf))
	}
	return nil
}

// InsertPDF appends every page of the source PDF (given as bytes) to this
// document. password unlocks an encrypted source ("" if none).
func (d *Document) InsertPDF(src []byte, password string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	if len(src) == 0 {
		return errors.New("gomupdf: empty source")
	}
	cdata := C.CBytes(src)
	defer C.free(cdata)
	cpw := C.CString(password)
	defer C.free(unsafe.Pointer(cpw))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_graft_bytes(d.ctx, d.doc, (*C.uchar)(cdata), C.size_t(len(src)), cpw, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: insert pdf: " + C.GoString(errBuf))
	}
	return nil
}

// InsertText draws a line of Helvetica text at (x, y) (baseline) on page i.
func (p *Page) InsertText(x, y, size float64, text string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	ct := C.CString(text)
	defer C.free(unsafe.Pointer(ct))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_insert_text(d.ctx, d.doc, C.int(p.Number), C.float(x), C.float(y), C.float(size), ct, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: insert text: " + C.GoString(errBuf))
	}
	return nil
}

// InsertImage places an encoded image (jpeg/png/…) into rect r on page i.
func (p *Page) InsertImage(r geometry.Rect, img []byte) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	if len(img) == 0 {
		return errors.New("gomupdf: empty image")
	}
	cdata := C.CBytes(img)
	defer C.free(cdata)
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_insert_image(d.ctx, d.doc, C.int(p.Number),
		C.float(r.X0), C.float(r.Y0), C.float(r.Width()), C.float(r.Height()),
		(*C.uchar)(cdata), C.size_t(len(img)), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: insert image: " + C.GoString(errBuf))
	}
	return nil
}

// AddRectAnnot adds a rectangle (square) annotation on page i.
func (p *Page) AddRectAnnot(r geometry.Rect) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_add_rect_annot(d.ctx, d.doc, C.int(p.Number), C.float(r.X0), C.float(r.Y0), C.float(r.X1), C.float(r.Y1), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: add annot: " + C.GoString(errBuf))
	}
	return nil
}

// imagesRaw returns the image-metadata trace output (IMG ... lines).
func (p *Page) imagesRaw() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_images(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: images: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

// imageBytesRaw returns the encoded bytes + extension of the index-th image.
func (p *Page) imageBytesRaw(index int) ([]byte, string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return nil, "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	extBuf := (*C.char)(C.malloc(8))
	defer C.free(unsafe.Pointer(extBuf))
	// Zero the first byte so that if the C side returns NULL without writing an
	// error message (the "no image at this index" case), GoString reads an empty
	// string rather than uninitialized heap memory.
	*errBuf = 0
	*extBuf = 0
	var outLen C.int
	ptr := C.gomupdf_image_bytes(d.ctx, d.doc, C.int(p.Number), C.int(index), &outLen, extBuf, errBuf, errBufLen)
	if ptr == nil {
		if msg := C.GoString(errBuf); msg != "" {
			return nil, "", errors.New("gomupdf: image bytes: " + msg)
		}
		return nil, "", nil // no image at that index
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), C.GoString(extBuf), nil
}

// pixmapRaw renders the page and returns the header+samples blob.
func (p *Page) pixmapRaw(zoom float64, gray, alpha bool) ([]byte, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	bi := func(b bool) C.int {
		if b {
			return 1
		}
		return 0
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	var outLen C.int
	ptr := C.gomupdf_pixmap(d.ctx, d.doc, C.int(p.Number), C.float(zoom), bi(gray), bi(alpha), &outLen, errBuf, errBufLen)
	if ptr == nil {
		return nil, errors.New("gomupdf: pixmap: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), nil
}

// drawingsRaw returns the trace-device output (P/m/l/c/h lines).
func (p *Page) drawingsRaw() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_drawings(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: drawings: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

// tocRaw returns the C outline output ("level\tpage\ttitle" per line).
func (d *Document) tocRaw() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_toc(d.ctx, d.doc, errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: toc: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

// linksRaw returns the C link output ("x0 y0 x1 y1\turi" per line).
func (p *Page) linksRaw() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_links(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: links: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

// searchRaw returns the C search output (one hit per line: 8 quad floats).
func (p *Page) searchRaw(needle string) (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	cn := C.CString(needle)
	defer C.free(unsafe.Pointer(cn))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_search(d.ctx, d.doc, C.int(p.Number), cn, errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: search: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

// wordsRaw returns the C word-extraction output (one word per line:
// "x0 y0 x1 y1 block line\t<text>"). Parsed by Page.Words.
func (p *Page) wordsRaw() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_words(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: words: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

// StructuredJSON returns MuPDF structured text as JSON (blocks → lines → chars,
// each with bbox). This is the raw geometry feed for layout reconstruction; see
// Words/Blocks for typed access.
func (p *Page) StructuredJSON() (string, error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_stext_json(d.ctx, d.doc, C.int(p.Number), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: stext json: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

const softHyphen = "­" // U+00AD

// Lines returns the page's non-empty lines with the soft hyphen (U+00AD)
// stripped and trailing whitespace trimmed.
func (p *Page) Lines() ([]string, error) {
	raw, err := p.GetText()
	if err != nil {
		return nil, err
	}
	return splitLines(raw), nil
}

func splitLines(raw string) []string {
	raw = strings.ReplaceAll(raw, softHyphen, "")
	var out []string
	for _, ln := range strings.Split(raw, "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		out = append(out, strings.TrimRight(ln, " \t\r"))
	}
	return out
}

// AllLines returns every page's Lines concatenated into a single flat line
// stream across the whole document.
func (d *Document) AllLines() ([]string, error) {
	var out []string
	for _, page := range d.Pages() {
		lines, err := page.Lines()
		if err != nil {
			return nil, err
		}
		out = append(out, lines...)
	}
	return out, nil
}

// Text returns the full document text, pages separated by a form feed ("\f").
func (d *Document) Text() (string, error) {
	var b strings.Builder
	for i, page := range d.Pages() {
		if i > 0 {
			b.WriteByte('\f')
		}
		t, err := page.GetText()
		if err != nil {
			return "", err
		}
		b.WriteString(t)
	}
	return b.String(), nil
}

// TextByPage returns each page's text as a slice.
func (d *Document) TextByPage() ([]string, error) {
	var out []string
	for _, page := range d.Pages() {
		t, err := page.GetText()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// metaKeys are the standard MuPDF metadata lookups exposed by Metadata.
var metaKeys = map[string]string{
	"format":       "format",
	"encryption":   "encryption",
	"title":        "info:Title",
	"author":       "info:Author",
	"subject":      "info:Subject",
	"keywords":     "info:Keywords",
	"creator":      "info:Creator",
	"producer":     "info:Producer",
	"creationDate": "info:CreationDate",
	"modDate":      "info:ModDate",
}

// Metadata returns document metadata (title, author, format, …). Absent keys are
// omitted.
func (d *Document) Metadata() (map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	const bufLen = 1024
	buf := (*C.char)(C.malloc(bufLen))
	defer C.free(unsafe.Pointer(buf))
	out := make(map[string]string)
	for name, key := range metaKeys {
		ck := C.CString(key)
		n := C.gomupdf_lookup_meta(d.ctx, d.doc, ck, buf, bufLen)
		C.free(unsafe.Pointer(ck))
		if n >= 0 {
			if v := strings.TrimSpace(C.GoString(buf)); v != "" {
				out[name] = v
			}
		}
	}
	return out, nil
}
