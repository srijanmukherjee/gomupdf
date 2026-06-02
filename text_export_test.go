package gomupdf

import (
	"strings"
	"testing"
)

// knownText is a substring expected in small-table.pdf page 0 (confirmed by
// TestTablesLinesSmallTable which asserts "Boiling Points" in row 0).
const knownTextExport = "Boiling"

func TestPageHTML(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}
	html, err := p.HTML()
	if err != nil {
		t.Fatalf("HTML: %v", err)
	}
	if html == "" {
		t.Fatal("HTML: got empty string")
	}
	if !strings.Contains(html, "<") {
		t.Errorf("HTML: expected at least one '<', got %q…", html[:clamp(len(html), 120)])
	}
	if !strings.Contains(html, "div") && !strings.Contains(html, "p") && !strings.Contains(html, "span") {
		t.Errorf("HTML: expected at least one of div/p/span; output starts: %q", html[:clamp(len(html), 200)])
	}
	if !strings.Contains(html, knownTextExport) {
		t.Errorf("HTML: expected %q in output; output starts: %q", knownTextExport, html[:clamp(len(html), 200)])
	}
}

func TestPageXHTML(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}
	xhtml, err := p.XHTML()
	if err != nil {
		t.Fatalf("XHTML: %v", err)
	}
	if xhtml == "" {
		t.Fatal("XHTML: got empty string")
	}
	if !strings.Contains(xhtml, "<") {
		t.Errorf("XHTML: expected at least one '<', got %q…", xhtml[:clamp(len(xhtml), 120)])
	}
	if !strings.Contains(xhtml, "p") && !strings.Contains(xhtml, "span") && !strings.Contains(xhtml, "div") {
		t.Errorf("XHTML: expected at least one markup tag; output starts: %q", xhtml[:clamp(len(xhtml), 200)])
	}
	if !strings.Contains(xhtml, knownTextExport) {
		t.Errorf("XHTML: expected %q in output; output starts: %q", knownTextExport, xhtml[:clamp(len(xhtml), 200)])
	}
}

func TestPageXML(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}
	xml, err := p.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	if xml == "" {
		t.Fatal("XML: got empty string")
	}
	if !strings.Contains(xml, "<") {
		t.Errorf("XML: expected at least one '<', got %q…", xml[:clamp(len(xml), 120)])
	}
	// MuPDF XML output uses <page>, <block>, <line>, <char> elements.
	if !strings.Contains(xml, "<page") && !strings.Contains(xml, "<block") && !strings.Contains(xml, "<char") {
		t.Errorf("XML: expected <page/<block/<char element; output starts: %q", xml[:clamp(len(xml), 200)])
	}
	// MuPDF's XML emits one <char c="X"/> element per glyph, so the known text
	// never appears as a contiguous substring. Assert its characters are present
	// as char attributes instead (robust across MuPDF versions).
	if !strings.Contains(xml, `c="B"`) || !strings.Contains(xml, `c="o"`) {
		t.Errorf("XML: expected per-character elements for %q; output starts: %q", knownTextExport, xml[:clamp(len(xml), 200)])
	}
}

func TestPageHTMLClosedDoc(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	p, _ := d.LoadPage(0)
	d.Close()
	_, err := p.HTML()
	if err == nil {
		t.Fatal("HTML: expected error on closed document, got nil")
	}
}

func TestPageXHTMLClosedDoc(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	p, _ := d.LoadPage(0)
	d.Close()
	_, err := p.XHTML()
	if err == nil {
		t.Fatal("XHTML: expected error on closed document, got nil")
	}
}

func TestPageXMLClosedDoc(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	p, _ := d.LoadPage(0)
	d.Close()
	_, err := p.XML()
	if err == nil {
		t.Fatal("XML: expected error on closed document, got nil")
	}
}

func clamp(a, b int) int {
	if a < b {
		return a
	}
	return b
}
