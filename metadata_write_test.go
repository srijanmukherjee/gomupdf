package gomupdf

import (
	"strings"
	"testing"
)

// SetMetadata values survive a save/reopen round-trip and are visible to the
// Metadata reader.
func TestSetMetadataRoundTrip(t *testing.T) {
	d, _ := NewPDF()
	_ = d.NewPage(200, 200)
	in := map[string]string{
		"title":    "Quarterly Report",
		"author":   "Srijan",
		"subject":  "Finance",
		"keywords": "paisa, pdf",
		"creator":  "gomupdf",
	}
	if err := d.SetMetadata(in); err != nil {
		d.Close()
		t.Fatal(err)
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatal(err)
	}

	d2, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	got, err := d2.Metadata()
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range in {
		if got[k] != want {
			t.Errorf("metadata[%q] = %q, want %q", k, got[k], want)
		}
	}
}

// SetMetadata works on a document opened from an existing file (not just
// freshly created ones), where the /Info dictionary may be absent.
func TestSetMetadataOpenedDoc(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	if err := d.SetMetadata(map[string]string{"title": "Opened", "author": "Test"}); err != nil {
		d.Close()
		t.Fatal(err)
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatal(err)
	}
	d2, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	got, _ := d2.Metadata()
	if got["title"] != "Opened" || got["author"] != "Test" {
		t.Errorf("opened-doc metadata = %+v, want title=Opened author=Test", got)
	}
}

// An empty value clears a previously-set field.
func TestSetMetadataClear(t *testing.T) {
	d, _ := NewPDF()
	_ = d.NewPage(200, 200)
	_ = d.SetMetadata(map[string]string{"title": "First"})
	if err := d.SetMetadata(map[string]string{"title": ""}); err != nil {
		d.Close()
		t.Fatal(err)
	}
	data, _ := d.SaveBytes(true)
	d.Close()
	d2, _ := OpenStream(data)
	defer d2.Close()
	got, _ := d2.Metadata()
	if v, ok := got["title"]; ok && v != "" {
		t.Errorf("title = %q, want cleared/absent", v)
	}
}

// Unknown keys are ignored rather than erroring.
func TestSetMetadataUnknownKeyIgnored(t *testing.T) {
	d, _ := NewPDF()
	defer d.Close()
	_ = d.NewPage(100, 100)
	if err := d.SetMetadata(map[string]string{"bogusKey": "x", "title": "T"}); err != nil {
		t.Fatalf("unknown key should be ignored, got %v", err)
	}
}

// XMP round-trips: set, save, reopen, read back the same packet, then delete.
func TestXMPRoundTrip(t *testing.T) {
	const packet = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` +
		`<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF ` +
		`xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>XMP Title</dc:title></rdf:Description></rdf:RDF></x:xmpmeta>` +
		`<?xpacket end="w"?>`

	d, _ := NewPDF()
	_ = d.NewPage(200, 200)
	if err := d.SetXMP(packet); err != nil {
		d.Close()
		t.Fatal(err)
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatal(err)
	}

	d2, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := d2.XMP()
	if err != nil {
		d2.Close()
		t.Fatal(err)
	}
	if !strings.Contains(got, "XMP Title") {
		t.Errorf("XMP round-trip lost content: %q", got)
	}

	// Now delete it and confirm it is gone after another round-trip.
	if err := d2.DeleteXMP(); err != nil {
		d2.Close()
		t.Fatal(err)
	}
	data2, _ := d2.SaveBytes(true)
	d2.Close()

	d3, _ := OpenStream(data2)
	defer d3.Close()
	got2, _ := d3.XMP()
	if strings.TrimSpace(got2) != "" {
		t.Errorf("XMP should be empty after delete, got %q", got2)
	}
}

// XMP on a document without one returns empty, not an error.
func TestXMPAbsent(t *testing.T) {
	d, _ := NewPDF()
	defer d.Close()
	_ = d.NewPage(100, 100)
	got, err := d.XMP()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("XMP = %q, want empty", got)
	}
}

// Metadata writes on a closed document error cleanly.
func TestSetMetadataClosedDoc(t *testing.T) {
	d, _ := NewPDF()
	d.Close()
	if err := d.SetMetadata(map[string]string{"title": "x"}); err == nil {
		t.Error("SetMetadata on closed doc should error")
	}
	if err := d.SetXMP("<x/>"); err == nil {
		t.Error("SetXMP on closed doc should error")
	}
}
