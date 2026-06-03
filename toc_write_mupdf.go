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

static int gomupdf_set_toc(fz_context *ctx, fz_document *doc,
                           char *spec, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) {
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));

        if (spec[0] == '\0') {
            pdf_dict_del(ctx, root, PDF_NAME(Outlines));
        } else {
            int n_pages = pdf_count_pages(ctx, pdf);

            pdf_obj *outlines = pdf_add_object_drop(ctx, pdf, pdf_new_dict(ctx, pdf, 4));
            pdf_dict_put_name(ctx, outlines, PDF_NAME(Type), "Outlines");

            pdf_obj *parent_stack[64];
            pdf_obj *last_child[64];
            int      child_count[64];
            memset(parent_stack, 0, sizeof(parent_stack));
            memset(last_child,   0, sizeof(last_child));
            memset(child_count,  0, sizeof(child_count));
            parent_stack[0] = outlines;

            for (char *line = strtok(spec, "\n"); line; line = strtok(NULL, "\n")) {
                char *p = line;
                char *e1 = strchr(p, '\t'); if (!e1) continue; *e1 = 0;
                int level = atoi(p); p = e1 + 1;
                if (level < 1) level = 1;
                if (level > 63) level = 63;
                char *e2 = strchr(p, '\t'); if (!e2) continue; *e2 = 0;
                int pageno = atoi(p); p = e2 + 1;
                char *title = p;

                if (pageno < 0) pageno = 0;
                if (pageno >= n_pages) pageno = n_pages - 1;

                pdf_obj *item = pdf_add_object_drop(ctx, pdf, pdf_new_dict(ctx, pdf, 8));

                pdf_dict_put_text_string(ctx, item, PDF_NAME(Title), title);

                pdf_obj *pageobj = pdf_lookup_page_obj(ctx, pdf, pageno);
                pdf_obj *dest = pdf_new_array(ctx, pdf, 2);
                pdf_array_push(ctx, dest, pageobj);
                pdf_array_push_drop(ctx, dest, PDF_NAME(Fit));
                pdf_dict_put_drop(ctx, item, PDF_NAME(Dest), dest);

                pdf_obj *parent = parent_stack[level - 1];
                pdf_dict_put(ctx, item, PDF_NAME(Parent), parent);

                if (last_child[level - 1] == NULL) {
                    pdf_dict_put(ctx, parent, PDF_NAME(First), item);
                } else {
                    pdf_dict_put(ctx, last_child[level - 1], PDF_NAME(Next), item);
                    pdf_dict_put(ctx, item, PDF_NAME(Prev), last_child[level - 1]);
                }
                pdf_dict_put(ctx, parent, PDF_NAME(Last), item);
                last_child[level - 1] = item;
                child_count[level - 1]++;

                parent_stack[level] = item;
                if (level < 63) {
                    last_child[level]   = NULL;
                    child_count[level]  = 0;
                }
            }

            for (int L = 0; L < 64; L++) {
                if (parent_stack[L] == NULL) break;
                if (child_count[L] > 0) {
                    pdf_dict_put_int(ctx, parent_stack[L], PDF_NAME(Count), child_count[L]);
                }
            }

            pdf_dict_put(ctx, root, PDF_NAME(Outlines), outlines);
        }
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func (d *mupdfDoc) setTOC(spec string) error {
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cspec := C.CString(spec)
	defer C.free(unsafe.Pointer(cspec))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_set_toc(d.ctx, d.doc, cspec, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set toc: " + C.GoString(errBuf))
	}
	return nil
}
