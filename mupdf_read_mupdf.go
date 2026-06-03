//go:build !nomupdf

package gomupdf

// Read-side MuPDF helpers that historically lived in mupdf.go: page text,
// words, bound, outline, links, search, images, drawings, structured JSON, and
// metadata lookup. Each is a *mupdfDoc method implementing docBackend.

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

// Extract reading-order plain text for one page (lines separated by '\n').
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

// Char-level word extraction. Emits one word per line as
// "x0 y0 x1 y1 blockno lineno\t<text>".
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

static int gomupdf_lookup_meta(fz_context *ctx, fz_document *doc, const char *key,
                               char *buf, int size) {
    int n = -1;
    fz_try(ctx) { n = fz_lookup_metadata(ctx, doc, key, buf, size); }
    fz_catch(ctx) { return -1; }
    return n;
}

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

// --- images trace device ---------------------------------------------------

static const char *gomupdf_image_ext(fz_context *ctx, fz_image *img) {
    fz_compressed_buffer *cb = fz_compressed_image_buffer(ctx, img);
    if (!cb) return "png";
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

// --- drawings trace device --------------------------------------------------

typedef struct {
    fz_context *ctx;
    fz_buffer *buf;
    fz_matrix ctm;
    float sx, sy;
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
*/
import "C"

import (
	"errors"
	"strings"
	"unsafe"
)

func (d *mupdfDoc) textRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_text(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: page text: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) wordsRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_words(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: words: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) boundRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_bound(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: bound: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) tocRaw() (string, error) {
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

func (d *mupdfDoc) linksRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_links(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: links: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) searchRaw(pageNo int, needle string) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	cn := C.CString(needle)
	defer C.free(unsafe.Pointer(cn))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_search(d.ctx, d.doc, C.int(pageNo), cn, errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: search: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) structuredJSON(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_page_stext_json(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: stext json: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) imagesRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_images(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: images: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) imageBytes(pageNo, index int) ([]byte, string, error) {
	if d.doc == nil {
		return nil, "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	extBuf := (*C.char)(C.malloc(8))
	defer C.free(unsafe.Pointer(extBuf))
	*errBuf = 0
	*extBuf = 0
	var outLen C.int
	ptr := C.gomupdf_image_bytes(d.ctx, d.doc, C.int(pageNo), C.int(index), &outLen, extBuf, errBuf, errBufLen)
	if ptr == nil {
		if msg := C.GoString(errBuf); msg != "" {
			return nil, "", errors.New("gomupdf: image bytes: " + msg)
		}
		return nil, "", nil // no image at that index
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), outLen), C.GoString(extBuf), nil
}

func (d *mupdfDoc) drawingsRaw(pageNo int) (string, error) {
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	cstr := C.gomupdf_drawings(d.ctx, d.doc, C.int(pageNo), errBuf, errBufLen)
	if cstr == nil {
		return "", errors.New("gomupdf: drawings: " + C.GoString(errBuf))
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), nil
}

func (d *mupdfDoc) lookupMeta(key string) (string, bool) {
	if d.doc == nil {
		return "", false
	}
	const bufLen = 1024
	buf := (*C.char)(C.malloc(bufLen))
	defer C.free(unsafe.Pointer(buf))
	ck := C.CString(key)
	defer C.free(unsafe.Pointer(ck))
	n := C.gomupdf_lookup_meta(d.ctx, d.doc, ck, buf, bufLen)
	if n < 0 {
		return "", false
	}
	return strings.TrimSpace(C.GoString(buf)), true
}
