package gomupdf

import (
	"errors"
	"os"
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

// SaveBytesWithOptions serializes the document to PDF bytes using opts.
func (d *Document) SaveBytesWithOptions(opts SaveOptions) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return nil, errors.New("gomupdf: document closed")
	}
	return d.b.save(opts, nil)
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
//
// Incremental save of an in-memory document requires a reasonably recent MuPDF;
// some older builds cannot relate the output to the original input and return
// "Cannot derive input stream from output stream".
func (d *Document) SaveIncremental(path string) error {
	return d.SaveWithOptions(path, SaveOptions{Incremental: true})
}
