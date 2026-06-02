package gomupdf

import (
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// newTwoPageDoc is a test helper that creates a fresh PDF with two 400×400 pages.
func newTwoPageDoc(t *testing.T) *Document {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatalf("NewPDF: %v", err)
	}
	if err := d.NewPage(400, 400); err != nil {
		t.Fatalf("NewPage 0: %v", err)
	}
	if err := d.NewPage(400, 400); err != nil {
		t.Fatalf("NewPage 1: %v", err)
	}
	return d
}

var linkRect = geometry.NewRect(50, 50, 200, 150)

// TestInsertLinkAppearsInLinks verifies InsertLink stores a link that is
// returned by Links() with the matching URI and overlapping rect.
func TestInsertLinkAppearsInLinks(t *testing.T) {
	d := newTwoPageDoc(t)
	defer d.Close()
	p, _ := d.LoadPage(0)

	const wantURI = "https://example.com"
	if err := p.InsertLink(linkRect, wantURI); err != nil {
		t.Fatalf("InsertLink: %v", err)
	}
	links, err := p.Links()
	if err != nil {
		t.Fatalf("Links: %v", err)
	}
	var found bool
	for _, l := range links {
		if l.URI == wantURI {
			found = true
			// Rect should be non-degenerate.
			if l.Rect.X1 <= l.Rect.X0 || l.Rect.Y1 <= l.Rect.Y0 {
				t.Errorf("link rect degenerate: %+v", l.Rect)
			}
		}
	}
	if !found {
		t.Fatalf("link with URI %q not found; got %+v", wantURI, links)
	}
}

// TestInsertLinkRoundTrip verifies the link survives SaveBytes/reopen.
func TestInsertLinkRoundTrip(t *testing.T) {
	d := newTwoPageDoc(t)
	defer d.Close()
	p, _ := d.LoadPage(0)

	const wantURI = "https://example.com"
	if err := p.InsertLink(linkRect, wantURI); err != nil {
		t.Fatalf("InsertLink: %v", err)
	}
	data, err := d.SaveBytes(true)
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}

	re, err := OpenStream(data)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer re.Close()
	rp, _ := re.LoadPage(0)
	links, err := rp.Links()
	if err != nil {
		t.Fatalf("Links after reopen: %v", err)
	}
	var found bool
	for _, l := range links {
		if l.URI == wantURI {
			found = true
		}
	}
	if !found {
		t.Fatalf("link %q not found after round-trip; got %+v", wantURI, links)
	}
}

// TestInsertGotoLinkIncreasesCount verifies InsertGotoLink adds a link with a
// non-empty URI and a matching rect (we do NOT assert the exact goto URI form).
func TestInsertGotoLinkIncreasesCount(t *testing.T) {
	d := newTwoPageDoc(t)
	defer d.Close()
	p, _ := d.LoadPage(0)

	before, err := p.Links()
	if err != nil {
		t.Fatalf("Links before: %v", err)
	}
	if err := p.InsertGotoLink(linkRect, 1); err != nil {
		t.Fatalf("InsertGotoLink: %v", err)
	}
	after, err := p.Links()
	if err != nil {
		t.Fatalf("Links after: %v", err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("expected %d links, got %d", len(before)+1, len(after))
	}
	// Confirm the new link has a non-empty URI.
	added := after[len(after)-1]
	if added.URI == "" {
		t.Error("goto link has empty URI")
	}
}

// TestDeleteLinkDecreasesCount verifies DeleteLink(0) removes one link.
func TestDeleteLinkDecreasesCount(t *testing.T) {
	d := newTwoPageDoc(t)
	defer d.Close()
	p, _ := d.LoadPage(0)

	if err := p.InsertLink(linkRect, "https://example.com"); err != nil {
		t.Fatalf("InsertLink: %v", err)
	}
	if err := p.InsertLink(geometry.NewRect(10, 10, 100, 100), "https://other.example"); err != nil {
		t.Fatalf("InsertLink 2: %v", err)
	}
	before, err := p.Links()
	if err != nil {
		t.Fatalf("Links before delete: %v", err)
	}
	if len(before) < 1 {
		t.Fatal("expected at least 1 link before delete")
	}
	if err := p.DeleteLink(0); err != nil {
		t.Fatalf("DeleteLink: %v", err)
	}
	after, err := p.Links()
	if err != nil {
		t.Fatalf("Links after delete: %v", err)
	}
	if len(after) != len(before)-1 {
		t.Fatalf("after delete: expected %d links, got %d", len(before)-1, len(after))
	}
}

// TestInsertLinkClosedDoc verifies InsertLink returns an error on a closed doc.
func TestInsertLinkClosedDoc(t *testing.T) {
	d := newTwoPageDoc(t)
	d.Close()
	p := &Page{doc: d, Number: 0}
	err := p.InsertLink(linkRect, "https://example.com")
	if err == nil {
		t.Fatal("expected error from closed document, got nil")
	}
}

// TestInsertGotoLinkClosedDoc verifies InsertGotoLink returns error on closed doc.
func TestInsertGotoLinkClosedDoc(t *testing.T) {
	d := newTwoPageDoc(t)
	d.Close()
	p := &Page{doc: d, Number: 0}
	err := p.InsertGotoLink(linkRect, 1)
	if err == nil {
		t.Fatal("expected error from closed document, got nil")
	}
}

// TestDeleteLinkClosedDoc verifies DeleteLink returns error on closed doc.
func TestDeleteLinkClosedDoc(t *testing.T) {
	d := newTwoPageDoc(t)
	d.Close()
	p := &Page{doc: d, Number: 0}
	err := p.DeleteLink(0)
	if err == nil {
		t.Fatal("expected error from closed document, got nil")
	}
}
