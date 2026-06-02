package gomupdf

import (
	"testing"
)

// TestGetFontsSmallTable verifies that GetFonts returns at least one font with
// a non-empty Name and Type on a page that is known to reference fonts.
func TestGetFontsSmallTable(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()

	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	fonts, err := p.GetFonts()
	if err != nil {
		t.Fatalf("GetFonts: %v", err)
	}
	if len(fonts) == 0 {
		t.Fatal("expected at least one font, got none")
	}
	for _, f := range fonts {
		if f.Name == "" {
			t.Errorf("font xref=%d has empty Name", f.Xref)
		}
		if f.Type == "" {
			t.Errorf("font xref=%d has empty Type", f.Xref)
		}
		if f.Xref <= 0 {
			t.Errorf("font %q has non-positive Xref=%d", f.Name, f.Xref)
		}
		t.Logf("font xref=%d name=%q type=%q encoding=%q embedded=%v",
			f.Xref, f.Name, f.Type, f.Encoding, f.Embedded)
	}
}

// TestGetFontsEmbedded verifies that GetFonts correctly reports Embedded==true
// for a PDF known to have embedded TrueType fonts (test_3179.pdf).
// It also verifies ExtractFont returns non-empty data and a recognised extension.
func TestGetFontsEmbedded(t *testing.T) {
	d := openFixture(t, "test_3179.pdf")
	defer d.Close()

	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	fonts, err := p.GetFonts()
	if err != nil {
		t.Fatalf("GetFonts: %v", err)
	}
	if len(fonts) == 0 {
		t.Fatal("expected at least one font")
	}

	var embedded []FontInfo
	for _, f := range fonts {
		t.Logf("font xref=%d name=%q type=%q embedded=%v", f.Xref, f.Name, f.Type, f.Embedded)
		if f.Embedded {
			embedded = append(embedded, f)
		}
	}

	if len(embedded) == 0 {
		t.Fatal("expected at least one embedded font on page 0 of test_3179.pdf")
	}

	// Pick the first embedded font and verify ExtractFont.
	ef := embedded[0]
	name, ext, data, err := d.ExtractFont(ef.Xref)
	if err != nil {
		t.Fatalf("ExtractFont(xref=%d): %v", ef.Xref, err)
	}
	if len(data) == 0 {
		t.Fatalf("ExtractFont(xref=%d) returned empty data for embedded font", ef.Xref)
	}
	if ext == "" {
		t.Errorf("ExtractFont(xref=%d) returned empty extension for embedded font", ef.Xref)
	}
	if name == "" {
		t.Errorf("ExtractFont(xref=%d) returned empty name", ef.Xref)
	}
	validExts := map[string]bool{"ttf": true, "cff": true, "otf": true, "pfa": true}
	if !validExts[ext] {
		t.Errorf("ExtractFont(xref=%d) returned unexpected extension %q", ef.Xref, ext)
	}
	t.Logf("ExtractFont: name=%q ext=%q bytes=%d", name, ext, len(data))
}

// TestExtractFontNonEmbedded verifies that ExtractFont returns empty data and
// no error for a font that is not embedded (Helvetica-like base font).
func TestExtractFontNonEmbedded(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()

	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}

	fonts, err := p.GetFonts()
	if err != nil {
		t.Fatalf("GetFonts: %v", err)
	}

	var nonEmbedded *FontInfo
	for i := range fonts {
		if !fonts[i].Embedded && fonts[i].Xref > 0 {
			nonEmbedded = &fonts[i]
			break
		}
	}
	if nonEmbedded == nil {
		t.Skip("no non-embedded font with valid xref found in small-table.pdf")
	}

	_, _, data, err := d.ExtractFont(nonEmbedded.Xref)
	if err != nil {
		t.Fatalf("ExtractFont on non-embedded font: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("ExtractFont on non-embedded font returned %d bytes, want 0", len(data))
	}
	t.Logf("ExtractFont non-embedded xref=%d: empty data (expected)", nonEmbedded.Xref)
}

// TestGetFontsClosedDocument verifies that GetFonts on a closed document returns an error.
func TestGetFontsClosedDocument(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	p, err := d.LoadPage(0)
	if err != nil {
		d.Close()
		t.Fatalf("LoadPage: %v", err)
	}
	d.Close()

	_, err = p.GetFonts()
	if err == nil {
		t.Fatal("expected error from GetFonts on closed document, got nil")
	}
}

// TestExtractFontClosedDocument verifies that ExtractFont on a closed document returns an error.
func TestExtractFontClosedDocument(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	d.Close()

	_, _, _, err := d.ExtractFont(1)
	if err == nil {
		t.Fatal("expected error from ExtractFont on closed document, got nil")
	}
}
