package gomupdf

import (
	"errors"
	"strings"
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
//
// Writing metadata into a document opened from an existing file requires a
// reasonably recent MuPDF; some older builds reject it ("not a dict (null)").
// Documents created with NewPDF are unaffected.
func (d *Document) SetMetadata(meta map[string]string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
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
	return d.b.setMetadata(b.String())
}

// XMP returns the document's XMP metadata packet (the XML in the Catalog's
// /Metadata stream), or the empty string when the document has none.
func (d *Document) XMP() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return "", errors.New("gomupdf: document closed")
	}
	data, err := d.b.xmp()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SetXMP installs xml as the document's XMP metadata packet, replacing any
// existing one. Changes take effect on the next Save.
func (d *Document) SetXMP(xml string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.setXMP([]byte(xml))
}

// DeleteXMP removes the document's XMP metadata stream, if any. It is a no-op
// when none is present. Changes take effect on the next Save.
func (d *Document) DeleteXMP() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.deleteXMP()
}
