package experimental_test

import (
	"os"
	"strings"
	"testing"

	exp "github.com/srijanmukherjee/gomupdf/experimental"
)

const sample = "../testdata/resources/small-table.pdf"

func TestOpenAndPages(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer d.Close()
	if d.NumPages() < 1 {
		t.Fatalf("page count = %d", d.NumPages())
	}
}

func TestOpenFromBytes(t *testing.T) {
	b, err := os.ReadFile(sample)
	if err != nil {
		t.Fatal(err)
	}
	d, err := exp.Open(b)
	if err != nil {
		t.Fatalf("open bytes: %v", err)
	}
	defer d.Close()
	if d.NumPages() < 1 {
		t.Fatalf("page count = %d", d.NumPages())
	}
}

func TestUnneededPasswordIsHarmless(t *testing.T) {
	// An unencrypted document opened with a password must succeed: NeedsPass is
	// false, so authentication is never attempted. Pins the behavior that callers
	// may always pass a password even when the document turns out to be unencrypted.
	d, err := exp.Open(sample, exp.Password("not-needed"))
	if err != nil {
		t.Fatalf("open unencrypted with password: %v", err)
	}
	defer d.Close()
	if d.NumPages() < 1 {
		t.Fatalf("page count = %d", d.NumPages())
	}
}

func TestWordsAndRows(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.Page(0)
	if err != nil {
		t.Fatal(err)
	}

	words, err := p.Words()
	if err != nil {
		t.Fatal(err)
	}
	if len(words) == 0 {
		t.Fatal("no words extracted")
	}
	for _, w := range words {
		if w.Rect.Empty() {
			t.Errorf("degenerate word box: %q %+v", w.Text, w.Rect)
		}
		if w.Height() <= 0 {
			t.Errorf("word %q has non-positive height", w.Text)
		}
	}

	rows, err := p.Rows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no rows clustered")
	}
	// Each row's words must be sorted left-to-right.
	for _, row := range rows {
		var prev float64 = -1e9
		for _, w := range row.Words {
			if w.Rect.X0 < prev {
				t.Errorf("row words not x-sorted: %s", row.Text())
			}
			prev = w.Rect.X0
		}
	}
}

func TestTextInRegion(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Bound()
	if err != nil {
		t.Fatal(err)
	}
	// Cropping the full page region should yield the same words as the page.
	all, err := p.TextIn(b)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(all) == "" {
		t.Fatal("TextIn(full page) empty")
	}
	// Top half should be a subset (no more text than the whole page).
	topHalf := exp.R(b.X0, b.Y0, b.X1, b.CenterY())
	top, err := p.TextIn(topHalf)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) > len(all) {
		t.Errorf("top-half text longer than full page (%d > %d)", len(top), len(all))
	}
}

func TestFindLocated(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	// Find any word of 3+ letters; each hit must carry a non-empty rect.
	hits, err := p.Find(`[A-Za-z]{3,}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Skip("no alphabetic content in sample")
	}
	for _, h := range hits {
		if h.Rect.Empty() {
			t.Errorf("located match %q has empty rect", h.Text)
		}
		if h.Context() == "" {
			t.Errorf("match %q has empty context", h.Text)
		}
	}
}

func TestValueRightOf(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	// Row: "Noble gases -269 -62 -170.5" — value right of the label.
	v, ok, err := p.ValueRightOf("Noble gases")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("label not found")
	}
	if v != "-269 -62 -170.5" {
		t.Errorf("ValueRightOf = %q, want %q", v, "-269 -62 -170.5")
	}

	if _, ok, _ := p.ValueRightOf("Nonexistent Label"); ok {
		t.Error("found a label that does not exist")
	}
}

func TestValueBelow(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	// "min" heads a column; the row below ("Noble gases ...") has -269 under it.
	v, ok, err := p.ValueBelow("min")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || v == "" {
		t.Fatalf("ValueBelow(min) = %q, ok=%v", v, ok)
	}
}

func TestInfo(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	info, err := d.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info.Pages < 1 {
		t.Errorf("Info.Pages = %d", info.Pages)
	}
	if info.Format == "" {
		t.Error("Info.Format empty")
	}
}

func TestTables(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	tabs, err := p.Tables()
	if err != nil {
		t.Fatal(err)
	}
	if len(tabs) == 0 {
		t.Fatal("no tables detected in small-table.pdf")
	}
	tbl := tabs[0]
	if tbl.NumRows() < 2 || tbl.NumCols() < 2 {
		t.Errorf("table shape rows=%d cols=%d", tbl.NumRows(), tbl.NumCols())
	}
	if tbl.Region().Empty() {
		t.Error("table region empty")
	}
}

func TestRenderPNG(t *testing.T) {
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	png, err := p.PNG(exp.DPI(72))
	if err != nil {
		t.Fatal(err)
	}
	if len(png) < 8 || string(png[1:4]) != "PNG" {
		t.Errorf("not PNG output (%d bytes)", len(png))
	}
}

func TestImagesExtract(t *testing.T) {
	const imgPDF = "../testdata/resources/image-file1.pdf"
	d, err := exp.Open(imgPDF)
	if err != nil {
		t.Skipf("open %s: %v", imgPDF, err)
	}
	defer d.Close()
	imgs, err := d.Images()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) == 0 {
		t.Skip("no embedded images in fixture")
	}
	for _, im := range imgs {
		if len(im.Bytes) == 0 || im.Ext == "" {
			t.Errorf("bad extracted image: ext=%q bytes=%d", im.Ext, len(im.Bytes))
		}
	}
}

func TestMergePDFs(t *testing.T) {
	out := t.TempDir() + "/merged.pdf"
	if err := exp.Merge(out, sample, sample); err != nil {
		t.Fatal(err)
	}
	d1, err := exp.Open(sample)
	if err != nil {
		t.Fatal(err)
	}
	one := d1.NumPages()
	d1.Close()

	dm, err := exp.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer dm.Close()
	if dm.NumPages() != one*2 {
		t.Errorf("merged pages = %d, want %d", dm.NumPages(), one*2)
	}
}

func TestAppendImage(t *testing.T) {
	// Pull a real encoded image out of a fixture, then append it as a page.
	src, err := exp.Open("../testdata/resources/image-file1.pdf")
	if err != nil {
		t.Skipf("open image fixture: %v", err)
	}
	imgs, err := src.Images()
	src.Close()
	if err != nil || len(imgs) == 0 {
		t.Skip("no extractable image in fixture")
	}

	doc, err := exp.NewDoc()
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	if err := doc.AppendImage(imgs[0].Bytes); err != nil {
		t.Fatalf("AppendImage: %v", err)
	}
	if doc.NumPages() != 1 {
		t.Errorf("pages after AppendImage = %d, want 1", doc.NumPages())
	}
	if _, err := doc.Bytes(); err != nil {
		t.Errorf("serialize: %v", err)
	}
}

func TestClusterFloats(t *testing.T) {
	// Two clear lanes at ~10 and ~100, tolerance 5.
	in := []float64{100, 9, 11, 101, 10, 99}
	got := exp.ClusterFloats(in, 5)
	if len(got) != 2 {
		t.Fatalf("want 2 clusters, got %d: %v", len(got), got)
	}
	if c := exp.Mean(got[0]); c < 8 || c > 12 {
		t.Errorf("lane 0 center = %v, want ~10", c)
	}
	if c := exp.Mean(got[1]); c < 98 || c > 102 {
		t.Errorf("lane 1 center = %v, want ~100", c)
	}
}
