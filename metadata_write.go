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

// Write standard info-dictionary fields from a spec of "Field\tValue" lines
// (an empty value deletes the field) in a SINGLE pass. The trailer /Info dict
// is fetched or created once and all fields are written into that same object.
// Doing every field in one cgo call is deliberate: on documents opened from a
// stream, a /Info put made in one call is not always visible to a pdf_dict_get
// in a later call (and older MuPDF builds mishandle /Info via fz_set_metadata
// entirely), so per-field calls would lose all but the last field.
static int gomupdf_set_metadata(fz_context *ctx, fz_document *doc, char *spec,
                                char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) {
        // Ensure /Info exists as a dict so older MuPDF builds don't reject the
        // write, then delegate each field to fz_set_metadata, which knows how
        // to mutate the trailer for both freshly created and stream-opened
        // documents (and tracks the change for incremental saves).
        pdf_obj *trailer = pdf_trailer(ctx, pdf);
        if (!pdf_is_dict(ctx, pdf_resolve_indirect(ctx, pdf_dict_get(ctx, trailer, PDF_NAME(Info))))) {
            pdf_obj *ref = pdf_add_object_drop(ctx, pdf, pdf_new_dict(ctx, pdf, 8));
            pdf_dict_put(ctx, trailer, PDF_NAME(Info), ref);
            pdf_drop_obj(ctx, ref);
        }
        for (char *line = strtok(spec, "\n"); line; line = strtok(NULL, "\n")) {
            char *tab = strchr(line, '\t');
            if (!tab) continue;
            *tab = 0;
            char *field = line;
            char *value = tab + 1; // remainder of the line, possibly empty
            char key[128];
            snprintf(key, sizeof(key), "info:%s", field);
            fz_set_metadata(ctx, doc, key, value);
        }
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

// Read the Catalog /Metadata stream (XMP). Returns malloc'd bytes + *out_len,
// or NULL with *out_len == 0 when no XMP stream exists (and no error).
static unsigned char *gomupdf_get_xmp(fz_context *ctx, fz_document *doc,
                                      int *out_len, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return NULL; }
    fz_buffer *buf = NULL;
    unsigned char *result = NULL;
    *out_len = 0;
    fz_var(buf);
    fz_try(ctx) {
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
        pdf_obj *md = pdf_dict_get(ctx, root, PDF_NAME(Metadata));
        if (md && pdf_is_stream(ctx, md)) {
            buf = pdf_load_stream(ctx, md);
            unsigned char *data = NULL;
            size_t n = fz_buffer_storage(ctx, buf, &data);
            result = (unsigned char *)malloc(n ? n : 1);
            if (result) { memcpy(result, data, n); *out_len = (int)n; }
        }
    }
    fz_always(ctx) { fz_drop_buffer(ctx, buf); }
    fz_catch(ctx) {
        if (result) { free(result); result = NULL; }
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        *out_len = -1;
        return NULL;
    }
    return result;
}

static int gomupdf_set_xmp(fz_context *ctx, fz_document *doc,
                           const unsigned char *data, size_t len,
                           char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_buffer *buf = NULL;
    pdf_obj *dict = NULL;
    pdf_obj *ref = NULL;
    fz_var(buf);
    fz_var(dict);
    fz_var(ref);
    fz_try(ctx) {
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
        buf = fz_new_buffer_from_copied_data(ctx, data, len);
        dict = pdf_new_dict(ctx, pdf, 2);
        pdf_dict_put(ctx, dict, PDF_NAME(Type), PDF_NAME(Metadata));
        pdf_dict_put(ctx, dict, PDF_NAME(Subtype), PDF_NAME(XML));
        ref = pdf_add_object(ctx, pdf, dict);
        pdf_update_stream(ctx, pdf, ref, buf, 0);
        pdf_dict_put(ctx, root, PDF_NAME(Metadata), ref);
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, ref);
        pdf_drop_obj(ctx, dict);
        fz_drop_buffer(ctx, buf);
    }
    fz_catch(ctx) {
        snprintf(err, errlen, "%s", fz_caught_message(ctx));
        return -1;
    }
    return 0;
}

static int gomupdf_del_xmp(fz_context *ctx, fz_document *doc, char *err, int errlen) {
    pdf_document *pdf = pdf_specifics(ctx, doc);
    if (!pdf) { snprintf(err, errlen, "not a PDF document"); return -1; }
    fz_try(ctx) {
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, pdf), PDF_NAME(Root));
        pdf_dict_del(ctx, root, PDF_NAME(Metadata));
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
	"strings"
	"unsafe"
)

// Metadata writing: the standard document info dictionary (title, author, …)
// and the XMP metadata stream. Reading is provided by Document.Metadata; these
// add the write side, mirroring PyMuPDF's set_metadata / set_xml_metadata /
// del_xml_metadata.

// writableMetaKeys maps the short metadata names accepted by SetMetadata to the
// MuPDF info-dictionary keys. The read-only "format" and "encryption" keys are
// intentionally excluded.
var writableMetaKeys = map[string]string{
	"title":        "info:Title",
	"author":       "info:Author",
	"subject":      "info:Subject",
	"keywords":     "info:Keywords",
	"creator":      "info:Creator",
	"producer":     "info:Producer",
	"creationDate": "info:CreationDate",
	"modDate":      "info:ModDate",
}

// SetMetadata writes standard document metadata. Keys use the same short names
// as Metadata returns ("title", "author", "subject", "keywords", "creator",
// "producer", "creationDate", "modDate"); unknown keys are ignored. An empty
// value clears that field. Changes take effect on the next Save.
func (d *Document) SetMetadata(meta map[string]string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	// Build a "Field\tValue" spec for the recognized keys; tabs and newlines in
	// values are flattened to spaces so the line format stays unambiguous.
	clean := strings.NewReplacer("\t", " ", "\n", " ")
	var b strings.Builder
	for name, value := range meta {
		key, ok := writableMetaKeys[name]
		if !ok {
			continue
		}
		b.WriteString(strings.TrimPrefix(key, "info:"))
		b.WriteByte('\t')
		b.WriteString(clean.Replace(value))
		b.WriteByte('\n')
	}
	if b.Len() == 0 {
		return nil
	}
	cspec := C.CString(b.String())
	defer C.free(unsafe.Pointer(cspec))
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_set_metadata(d.ctx, d.doc, cspec, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set metadata: " + C.GoString(errBuf))
	}
	return nil
}

// XMP returns the document's XMP metadata packet (the XML in the Catalog's
// /Metadata stream), or the empty string when the document has none.
func (d *Document) XMP() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return "", errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	var outLen C.int
	ptr := C.gomupdf_get_xmp(d.ctx, d.doc, &outLen, errBuf, errBufLen)
	if ptr == nil {
		if outLen < 0 {
			return "", errors.New("gomupdf: xmp: " + C.GoString(errBuf))
		}
		return "", nil // no XMP stream
	}
	defer C.free(unsafe.Pointer(ptr))
	return string(C.GoBytes(unsafe.Pointer(ptr), outLen)), nil
}

// SetXMP installs xml as the document's XMP metadata packet, replacing any
// existing one. Changes take effect on the next Save.
func (d *Document) SetXMP(xml string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	cdata := C.CBytes([]byte(xml))
	defer C.free(cdata)
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_set_xmp(d.ctx, d.doc, (*C.uchar)(cdata), C.size_t(len(xml)), errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: set xmp: " + C.GoString(errBuf))
	}
	return nil
}

// DeleteXMP removes the document's XMP metadata stream, if any. It is a no-op
// when none is present. Changes take effect on the next Save.
func (d *Document) DeleteXMP() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.doc == nil {
		return errors.New("gomupdf: document closed")
	}
	errBuf := (*C.char)(C.malloc(errBufLen))
	defer C.free(unsafe.Pointer(errBuf))
	if C.gomupdf_del_xmp(d.ctx, d.doc, errBuf, errBufLen) != 0 {
		return errors.New("gomupdf: delete xmp: " + C.GoString(errBuf))
	}
	return nil
}
