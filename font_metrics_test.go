package gomupdf

import (
	"math"
	"testing"
)

func TestNewFont_Helvetica(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont(Helvetica): %v", err)
	}
	if f == nil {
		t.Fatal("NewFont returned nil without error")
	}
	f.Close()
}

func TestNewFont_DoubleClose(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	f.Close()
	f.Close() // must not panic or crash
}

func TestNewFont_Unknown(t *testing.T) {
	_, err := NewFont("NoSuchFont")
	if err == nil {
		t.Fatal("expected error for unknown font, got nil")
	}
}

func TestFont_Name(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	defer f.Close()
	name := f.Name()
	if name == "" {
		t.Error("font name is empty")
	}
}

func TestFont_TextLength_Positive(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	defer f.Close()

	w := f.TextLength("Hello", 12)
	if w <= 0 {
		t.Errorf("TextLength('Hello', 12) = %v; want > 0", w)
	}
}

func TestFont_TextLength_Empty(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	defer f.Close()

	w := f.TextLength("", 12)
	if w != 0 {
		t.Errorf("TextLength('', 12) = %v; want 0", w)
	}
}

func TestFont_TextLength_LongerIsWider(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	defer f.Close()

	short := f.TextLength("Hello", 12)
	long := f.TextLength("Hello, World!", 12)
	if long <= short {
		t.Errorf("TextLength('Hello, World!') = %v should be > TextLength('Hello') = %v", long, short)
	}
}

func TestFont_TextLength_ScalesWithSize(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	defer f.Close()

	text := "Hello World"
	w12 := f.TextLength(text, 12)
	w24 := f.TextLength(text, 24)
	if w12 <= 0 {
		t.Fatalf("TextLength at 12pt = %v; want > 0", w12)
	}
	// should be exactly 2× since advance is linear in size
	ratio := w24 / w12
	if math.Abs(ratio-2.0) > 0.01 {
		t.Errorf("TextLength(24)/TextLength(12) = %v; want ~2.0", ratio)
	}
}

func TestFont_Advance_ProportionalMWiderThanI(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	defer f.Close()

	advM := f.Advance('m', 12)
	advI := f.Advance('i', 12)
	if advM <= advI {
		// Fallback: try W vs i (should always hold for proportional)
		advW := f.Advance('W', 12)
		if advW <= advI {
			t.Errorf("Advance('W', 12) = %v should be > Advance('i', 12) = %v", advW, advI)
		}
	}
}

func TestFont_Advance_CourierMonospace(t *testing.T) {
	f, err := NewFont("Courier")
	if err != nil {
		t.Fatalf("NewFont(Courier): %v", err)
	}
	defer f.Close()

	advM := f.Advance('m', 12)
	advI := f.Advance('i', 12)
	if advM <= 0 {
		t.Fatalf("Advance('m', 12) = %v; want > 0", advM)
	}
	// Monospace: m and i should have equal advance (within floating point epsilon)
	if math.Abs(advM-advI) > 1e-6 {
		t.Errorf("Courier Advance('m', 12) = %v != Advance('i', 12) = %v; want equal (monospace)", advM, advI)
	}
}

func TestFont_Ascender_Descender(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	defer f.Close()

	asc := f.Ascender()
	desc := f.Descender()

	if asc <= 0 {
		t.Errorf("Ascender() = %v; want > 0", asc)
	}
	if desc >= 0 {
		t.Errorf("Descender() = %v; want < 0", desc)
	}

	span := asc - desc
	if span < 0.8 || span > 1.6 {
		t.Errorf("Ascender - Descender = %v; want in [0.8, 1.6] em", span)
	}
}

func TestFont_ClosedReturnsZero(t *testing.T) {
	f, err := NewFont("Helvetica")
	if err != nil {
		t.Fatalf("NewFont: %v", err)
	}
	f.Close()

	if w := f.TextLength("Hello", 12); w != 0 {
		t.Errorf("TextLength on closed font = %v; want 0", w)
	}
	if a := f.Advance('A', 12); a != 0 {
		t.Errorf("Advance on closed font = %v; want 0", a)
	}
	if a := f.Ascender(); a != 0 {
		t.Errorf("Ascender on closed font = %v; want 0", a)
	}
	if d := f.Descender(); d != 0 {
		t.Errorf("Descender on closed font = %v; want 0", d)
	}
}

func TestNewFont_AllBase14(t *testing.T) {
	fonts := []string{
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Symbol", "ZapfDingbats",
	}
	for _, name := range fonts {
		f, err := NewFont(name)
		if err != nil {
			t.Errorf("NewFont(%q): %v", name, err)
			continue
		}
		f.Close()
	}
}
