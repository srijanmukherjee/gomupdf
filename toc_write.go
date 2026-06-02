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

// Build the /Outlines tree from a spec string: one entry per line as
// "level\tpage\ttitle" (title is the remainder; tabs/newlines stripped).
// An empty spec removes /Outlines entirely.
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

            // outlines is an indirect object so cross-references work.
            pdf_obj *outlines = pdf_add_object_drop(ctx, pdf, pdf_new_dict(ctx, pdf, 4));
            pdf_dict_put_name(ctx, outlines, PDF_NAME(Type), "Outlines");

            // parent_stack[L] is the parent node for items at level L+1.
            // parent_stack[0] is always the outlines root.
            pdf_obj *parent_stack[64];
            pdf_obj *last_child[64]; // last direct child added to parent_stack[L]
            int      child_count[64]; // # direct children for each level
            memset(parent_stack, 0, sizeof(parent_stack));
            memset(last_child,   0, sizeof(last_child));
            memset(child_count,  0, sizeof(child_count));
            parent_stack[0] = outlines;

            for (char *line = strtok(spec, "\n"); line; line = strtok(NULL, "\n")) {
                char *p = line;
                // field 1: level
                char *e1 = strchr(p, '\t'); if (!e1) continue; *e1 = 0;
                int level = atoi(p); p = e1 + 1;
                if (level < 1) level = 1;
                if (level > 63) level = 63;
                // field 2: page
                char *e2 = strchr(p, '\t'); if (!e2) continue; *e2 = 0;
                int pageno = atoi(p); p = e2 + 1;
                // remainder: title
                char *title = p;

                // clamp page to [0, n_pages-1]
                if (pageno < 0) pageno = 0;
                if (pageno >= n_pages) pageno = n_pages - 1;

                // create item as indirect object
                pdf_obj *item = pdf_add_object_drop(ctx, pdf, pdf_new_dict(ctx, pdf, 8));

                // /Title
                pdf_dict_put_text_string(ctx, item, PDF_NAME(Title), title);

                // /Dest = [ <pageobj> /Fit ]
                pdf_obj *pageobj = pdf_lookup_page_obj(ctx, pdf, pageno);
                pdf_obj *dest = pdf_new_array(ctx, pdf, 2);
                pdf_array_push(ctx, dest, pageobj);
                pdf_array_push_drop(ctx, dest, PDF_NAME(Fit));
                pdf_dict_put_drop(ctx, item, PDF_NAME(Dest), dest);

                // parent is parent_stack[level-1]
                pdf_obj *parent = parent_stack[level - 1];
                pdf_dict_put(ctx, item, PDF_NAME(Parent), parent);

                // link prev/next siblings
                if (last_child[level - 1] == NULL) {
                    // first child of this parent
                    pdf_dict_put(ctx, parent, PDF_NAME(First), item);
                } else {
                    pdf_dict_put(ctx, last_child[level - 1], PDF_NAME(Next), item);
                    pdf_dict_put(ctx, item, PDF_NAME(Prev), last_child[level - 1]);
                }
                pdf_dict_put(ctx, parent, PDF_NAME(Last), item);
                last_child[level - 1] = item;
                child_count[level - 1]++;

                // this item becomes parent for deeper levels
                parent_stack[level] = item;
                // reset tracking for deeper levels
                if (level < 63) {
                    last_child[level]   = NULL;
                    child_count[level]  = 0;
                }
            }

            // Set /Count on outlines root and all items that have children.
            // /Count = positive integer = number of immediate open children.
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
	"strconv"
	"strings"
	"unsafe"
)

// SetTOC replaces the document outline (bookmarks) with the given flat,
// depth-first entries. Level is 1-based (1 = top level); each entry's Level
// must be <= previous level + 1. Page is 0-based; out-of-range pages clamp to
// the nearest valid page. Passing nil removes the outline. Effective on Save.
func (d *Document) SetTOC(entries []TOCEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}

	var b strings.Builder
	replacer := strings.NewReplacer("\t", " ", "\n", " ")
	for _, e := range entries {
		b.WriteString(strconv.Itoa(e.Level))
		b.WriteByte('\t')
		b.WriteString(strconv.Itoa(e.Page))
		b.WriteByte('\t')
		b.WriteString(replacer.Replace(e.Title))
		b.WriteByte('\n')
	}

	cspec := C.CString(b.String())
	defer C.free(unsafe.Pointer(cspec))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))

	if C.gomupdf_set_toc(d.ctx, d.doc, cspec, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set toc: " + C.GoString(errBuf))
	}
	return nil
}
