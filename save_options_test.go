package gomupdf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// buildDoc makes a small multi-page PDF with text for save tests.
func buildDoc(t *testing.T) *Document {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := d.NewPage(300, 300); err != nil {
			d.Close()
			t.Fatal(err)
		}
		p, _ := d.LoadPage(i)
		_ = p.InsertText(50, 150, 18, "Hello page")
	}
	return d
}

// Deflate produces smaller output than the uncompressed zero-value options, and
// both reopen cleanly with identical page counts.
func TestSaveDeflateShrinks(t *testing.T) {
	d := buildDoc(t)
	defer d.Close()

	plain, err := d.SaveBytesWithOptions(SaveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	deflated, err := d.SaveBytesWithOptions(SaveOptions{Garbage: 3, Deflate: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(deflated) >= len(plain) {
		t.Errorf("deflated size %d not smaller than plain %d", len(deflated), len(plain))
	}
	for _, b := range [][]byte{plain, deflated} {
		rd, err := OpenStream(b)
		if err != nil {
			t.Fatalf("reopen: %v", err)
		}
		if rd.PageCount() != 3 {
			t.Errorf("page count = %d, want 3", rd.PageCount())
		}
		rd.Close()
	}
}

// A PDF saved with Encrypt requires the password to reopen.
func TestSaveEncryptOption(t *testing.T) {
	d := buildDoc(t)
	defer d.Close()
	data, err := d.SaveBytesWithOptions(SaveOptions{
		Encrypt:      true,
		UserPassword: "s3cret",
		Permissions:  -1,
	})
	if err != nil {
		t.Fatal(err)
	}
	rd, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	if !rd.NeedsPass() {
		t.Fatal("encrypted doc should need a password")
	}
	if rd.Authenticate("wrong") {
		t.Error("wrong password should not authenticate")
	}
	if !rd.Authenticate("s3cret") {
		t.Error("correct password should authenticate")
	}
}

// countHighBytes counts bytes above 0x7F.
func countHighBytes(b []byte) int {
	n := 0
	for _, c := range b {
		if c > 0x7F {
			n++
		}
	}
	return n
}

// ASCII output escapes binary stream content, so it carries far fewer
// high bytes than a binary (deflated) save. MuPDF always emits a small
// 4-byte binary header comment, so the count is low but not exactly zero.
func TestSaveASCII(t *testing.T) {
	d := buildDoc(t)
	defer d.Close()
	// Deflated content streams are binary, giving real content to escape.
	binary, err := d.SaveBytesWithOptions(SaveOptions{Deflate: true})
	if err != nil {
		t.Fatal(err)
	}
	ascii, err := d.SaveBytesWithOptions(SaveOptions{ASCII: true, Deflate: true})
	if err != nil {
		t.Fatal(err)
	}
	hb := countHighBytes(ascii)
	if hb >= countHighBytes(binary) {
		t.Errorf("ASCII high-byte count %d not below binary count %d", hb, countHighBytes(binary))
	}
	if hb > 16 {
		t.Errorf("ASCII output has %d high bytes, expected only the small header comment", hb)
	}
}

// EzSave writes a reopenable file to disk.
func TestEzSave(t *testing.T) {
	d := buildDoc(t)
	defer d.Close()
	path := filepath.Join(t.TempDir(), "ez.pdf")
	if err := d.EzSave(path); err != nil {
		t.Fatal(err)
	}
	rd, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	if rd.PageCount() != 3 {
		t.Errorf("page count = %d, want 3", rd.PageCount())
	}
}

// SaveIncremental on a stream-opened document appends an update: the output
// starts with the original bytes and reopens with the modification visible.
func TestSaveIncremental(t *testing.T) {
	base := buildDoc(t)
	orig, err := base.SaveBytesWithOptions(SaveOptions{Garbage: 3, Deflate: true})
	base.Close()
	if err != nil {
		t.Fatal(err)
	}

	d, err := OpenStream(orig)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	// Mutate: set a title so the incremental section has something to write.
	if err := d.SetMetadata(map[string]string{"title": "Incremental"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "incr.pdf")
	if err := d.SaveIncremental(path); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, orig) {
		t.Error("incremental output should begin with the original bytes")
	}
	if len(out) <= len(orig) {
		t.Errorf("incremental output %d not larger than original %d", len(out), len(orig))
	}
	rd, _ := OpenStream(out)
	defer rd.Close()
	meta, _ := rd.Metadata()
	if meta["title"] != "Incremental" {
		t.Errorf("incremental update lost title: %q", meta["title"])
	}
}

// Incremental save is rejected for documents created in memory (no original).
func TestSaveIncrementalNoOriginal(t *testing.T) {
	d := buildDoc(t)
	defer d.Close()
	path := filepath.Join(t.TempDir(), "x.pdf")
	if err := d.SaveIncremental(path); err == nil {
		t.Error("incremental save on an in-memory document should error")
	}
}

// Save options on a closed document error cleanly.
func TestSaveOptionsClosedDoc(t *testing.T) {
	d, _ := NewPDF()
	d.Close()
	if _, err := d.SaveBytesWithOptions(DefaultSaveOptions()); err == nil {
		t.Error("save on closed doc should error")
	}
}
