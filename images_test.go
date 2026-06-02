package gomupdf

import "testing"

// Ported concepts from test_imagebbox.py / test_extractimage.py using
// image-file1.pdf, which places one 439×501 JPEG twice on page 0.
func TestGetImages(t *testing.T) {
	d := openFixture(t, "image-file1.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	imgs, err := p.GetImages()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 2 {
		t.Fatalf("want 2 images, got %d", len(imgs))
	}
	for _, im := range imgs {
		if im.Width != 439 || im.Height != 501 {
			t.Errorf("image dims = %dx%d, want 439x501", im.Width, im.Height)
		}
		if im.N != 3 || im.BPC != 8 || im.Ext != "jpeg" {
			t.Errorf("image meta n=%d bpc=%d ext=%s, want 3/8/jpeg", im.N, im.BPC, im.Ext)
		}
		if im.BBox.Width() <= 0 || im.BBox.Height() <= 0 {
			t.Errorf("degenerate bbox %+v", im.BBox)
		}
	}
	// The two placements must have different bounding boxes.
	if imgs[0].BBox.Equal(imgs[1].BBox) {
		t.Error("the two image placements should have distinct bboxes")
	}
}

func TestExtractImage(t *testing.T) {
	d := openFixture(t, "image-file1.pdf")
	defer d.Close()
	p, _ := d.LoadPage(0)
	ex, err := p.ExtractImage(0)
	if err != nil {
		t.Fatal(err)
	}
	if ex == nil {
		t.Fatal("no image at index 0")
	}
	if ex.Ext != "jpeg" {
		t.Errorf("ext = %s, want jpeg", ex.Ext)
	}
	// JPEG magic FF D8 FF.
	if len(ex.Bytes) < 3 || ex.Bytes[0] != 0xFF || ex.Bytes[1] != 0xD8 || ex.Bytes[2] != 0xFF {
		t.Errorf("expected JPEG SOI marker, got % x", ex.Bytes[:min3(len(ex.Bytes), 3)])
	}
	// Out-of-range index returns nil, no error.
	none, err := p.ExtractImage(99)
	if err != nil {
		t.Fatal(err)
	}
	if none != nil {
		t.Error("expected nil for out-of-range image index")
	}
}

func min3(a, b int) int {
	if a < b {
		return a
	}
	return b
}
