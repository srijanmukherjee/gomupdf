package gomupdf

import (
	"bytes"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// renderOpsFixture renders small-table.pdf page 0 with the given options.
func renderOpsFixture(t *testing.T, opts ...PixmapOptions) *Pixmap {
	t.Helper()
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("LoadPage: %v", err)
	}
	var pm *Pixmap
	if len(opts) > 0 {
		pm, err = p.Pixmap(opts[0])
	} else {
		pm, err = p.Pixmap(PixmapOptions{Zoom: 1})
	}
	if err != nil {
		t.Fatalf("Pixmap: %v", err)
	}
	return pm
}

// copyPixmap returns a deep copy of Samples.
func copyPixmap(pm *Pixmap) []byte {
	s := make([]byte, len(pm.Samples))
	copy(s, pm.Samples)
	return s
}

// TestInvertDoubleIsIdentity verifies that inverting twice returns the original samples.
func TestInvertDoubleIsIdentity(t *testing.T) {
	pm := renderOpsFixture(t)
	orig := copyPixmap(pm)
	pm.Invert()
	pm.Invert()
	for i, v := range pm.Samples {
		if v != orig[i] {
			t.Fatalf("double-invert mismatch at byte %d: got %d, want %d", i, v, orig[i])
		}
	}
}

// TestInvertChanges verifies that a single invert changes at least some samples.
func TestInvertChanges(t *testing.T) {
	pm := renderOpsFixture(t)
	orig := copyPixmap(pm)
	pm.Invert()
	changed := false
	for i, v := range pm.Samples {
		if v != orig[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("Invert did not change any samples")
	}
}

// TestGammaIdentityApprox verifies that Gamma(1.0) is approximately the identity (±1 rounding).
func TestGammaIdentityApprox(t *testing.T) {
	pm := renderOpsFixture(t)
	orig := copyPixmap(pm)
	pm.Gamma(1.0)
	for i, v := range pm.Samples {
		diff := int(v) - int(orig[i])
		if diff < -1 || diff > 1 {
			t.Fatalf("Gamma(1.0) diff > ±1 at byte %d: got %d want %d", i, v, orig[i])
		}
	}
}

// TestGammaChangesAndInRange verifies Gamma(2.0) changes samples and keeps them in [0,255].
func TestGammaChangesAndInRange(t *testing.T) {
	pm := renderOpsFixture(t)
	orig := copyPixmap(pm)
	pm.Gamma(2.0)
	changed := false
	for i, v := range pm.Samples {
		if v > 255 { // byte can't exceed 255 but check stays for clarity
			t.Fatalf("sample %d out of range: %d", i, v)
		}
		if v != orig[i] {
			changed = true
		}
	}
	if !changed {
		t.Fatal("Gamma(2.0) did not change any samples")
	}
}

// TestGammaZeroNoOp verifies that Gamma(0) is a no-op.
func TestGammaZeroNoOp(t *testing.T) {
	pm := renderOpsFixture(t)
	orig := copyPixmap(pm)
	pm.Gamma(0)
	for i, v := range pm.Samples {
		if v != orig[i] {
			t.Fatalf("Gamma(0) changed sample at byte %d: got %d want %d", i, v, orig[i])
		}
	}
}

// TestJPEGSOIMarker verifies JPEG output starts with 0xFF 0xD8 and decodes cleanly.
func TestJPEGSOIMarker(t *testing.T) {
	pm := renderOpsFixture(t)
	data, err := pm.JPEG(80)
	if err != nil {
		t.Fatalf("JPEG: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("JPEG returned empty bytes")
	}
	if data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatalf("JPEG SOI marker missing: got %02X %02X", data[0], data[1])
	}
	if _, err := jpeg.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("jpeg.Decode: %v", err)
	}
}

// TestPNMFormat verifies PNM output starts with P5 or P6 and has the correct length.
func TestPNMFormat(t *testing.T) {
	pm := renderOpsFixture(t)
	data, err := pm.PNM()
	if err != nil {
		t.Fatalf("PNM: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("PNM returned empty bytes")
	}

	var magic string
	var colorComps int
	if bytes.HasPrefix(data, []byte("P5")) {
		magic = "P5"
		colorComps = 1
	} else if bytes.HasPrefix(data, []byte("P6")) {
		magic = "P6"
		colorComps = 3
	} else {
		t.Fatalf("PNM magic not P5 or P6: %q", data[:2])
	}
	_ = magic

	// header: "P5\n%d %d\n255\n" or "P6\n..."
	// find end of header (after third newline)
	headerEnd := 0
	newlines := 0
	for i, b := range data {
		if b == '\n' {
			newlines++
			if newlines == 3 {
				headerEnd = i + 1
				break
			}
		}
	}
	pixelBytes := len(data) - headerEnd
	expected := pm.Width * pm.Height * colorComps
	if pixelBytes != expected {
		t.Fatalf("PNM pixel data length = %d, want %d (W=%d H=%d cc=%d)",
			pixelBytes, expected, pm.Width, pm.Height, colorComps)
	}
}

// TestBytesPNG verifies Bytes("png") == PNG().
func TestBytesPNG(t *testing.T) {
	pm := renderOpsFixture(t)
	want, err := pm.PNG()
	if err != nil {
		t.Fatalf("PNG: %v", err)
	}
	got, err := pm.Bytes("png")
	if err != nil {
		t.Fatalf("Bytes(png): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("Bytes(png) != PNG()")
	}
}

// TestBytesJPEGDecodes verifies Bytes("jpg") decodes without error.
func TestBytesJPEGDecodes(t *testing.T) {
	pm := renderOpsFixture(t)
	data, err := pm.Bytes("jpg")
	if err != nil {
		t.Fatalf("Bytes(jpg): %v", err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("jpeg.Decode: %v", err)
	}
}

// TestBytesBogusErrors verifies Bytes with unknown format returns an error.
func TestBytesBogusErrors(t *testing.T) {
	pm := renderOpsFixture(t)
	_, err := pm.Bytes("bogus")
	if err == nil {
		t.Fatal("Bytes(bogus) expected error, got nil")
	}
}

// TestSaveFormats verifies Save writes non-empty files for .png, .jpg, .pnm.
func TestSaveFormats(t *testing.T) {
	pm := renderOpsFixture(t)
	dir := t.TempDir()

	for _, ext := range []string{".png", ".jpg", ".pnm"} {
		path := filepath.Join(dir, "out"+ext)
		if err := pm.Save(path); err != nil {
			t.Errorf("Save(%s): %v", ext, err)
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("stat %s: %v", path, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("Save(%s) wrote empty file", ext)
		}
	}
}

// TestSaveUnknownExtErrors verifies Save with an unknown extension returns an error.
func TestSaveUnknownExtErrors(t *testing.T) {
	pm := renderOpsFixture(t)
	dir := t.TempDir()
	err := pm.Save(filepath.Join(dir, "out.xyz"))
	if err == nil {
		t.Fatal("Save(.xyz) expected error, got nil")
	}
}

// TestInvertPreservesAlpha verifies that Invert leaves alpha bytes untouched.
func TestInvertPreservesAlpha(t *testing.T) {
	pm := renderOpsFixture(t, PixmapOptions{Zoom: 1, Alpha: true})
	if !pm.Alpha || pm.N < 2 {
		t.Skip("pixmap does not have alpha")
	}

	// Collect alpha bytes before invert.
	alphaIdx := pm.N - 1
	before := make([]byte, pm.Width*pm.Height)
	i := 0
	for y := 0; y < pm.Height; y++ {
		for x := 0; x < pm.Width; x++ {
			before[i] = pm.Samples[y*pm.Stride+x*pm.N+alphaIdx]
			i++
		}
	}

	pm.Invert()

	// Check alpha bytes after invert.
	i = 0
	for y := 0; y < pm.Height; y++ {
		for x := 0; x < pm.Width; x++ {
			got := pm.Samples[y*pm.Stride+x*pm.N+alphaIdx]
			if got != before[i] {
				t.Fatalf("alpha changed at (%d,%d): was %d, now %d", x, y, before[i], got)
			}
			i++
		}
	}
}
