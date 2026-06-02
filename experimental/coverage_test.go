package experimental_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/srijanmukherjee/gomupdf"
	exp "github.com/srijanmukherjee/gomupdf/experimental"
)

const imgSample = "../testdata/resources/image-file1.pdf"

// openSample opens the shared small-table fixture and registers cleanup.
func openSample(t *testing.T) *exp.Doc {
	t.Helper()
	d, err := exp.Open(sample)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(d.Close)
	return d
}

func page0(t *testing.T, d *exp.Doc) *exp.Page {
	t.Helper()
	p, err := d.Page(0)
	if err != nil {
		t.Fatalf("page 0: %v", err)
	}
	return p
}

// --- Rect helpers ---------------------------------------------------------

func TestRectHelpers(t *testing.T) {
	r := exp.R(0, 0, 10, 20)
	if r.Width() != 10 || r.Height() != 20 {
		t.Errorf("width/height = %v/%v", r.Width(), r.Height())
	}
	if r.CenterX() != 5 || r.CenterY() != 10 {
		t.Errorf("center = %v/%v", r.CenterX(), r.CenterY())
	}
	if r.Empty() {
		t.Error("non-degenerate rect reported empty")
	}
	if !(exp.Rect{}).Empty() {
		t.Error("zero rect should be empty")
	}

	inner := exp.R(1, 1, 5, 5)
	if !r.Contains(inner) {
		t.Error("Contains should be true for fully-enclosed rect")
	}
	if r.Contains(exp.R(-1, -1, 5, 5)) {
		t.Error("Contains should be false when partly outside")
	}

	if !r.ContainsPoint(5, 5) {
		t.Error("ContainsPoint(5,5) should be inside")
	}
	if r.ContainsPoint(100, 100) {
		t.Error("ContainsPoint(100,100) should be outside")
	}

	if !r.Overlaps(exp.R(5, 5, 30, 30)) {
		t.Error("Overlaps should detect intersection")
	}
	if r.Overlaps(exp.R(50, 50, 60, 60)) {
		t.Error("Overlaps should be false for disjoint rects")
	}

	u := exp.R(0, 0, 5, 5).Union(exp.R(10, 10, 20, 20))
	if u != exp.R(0, 0, 20, 20) {
		t.Errorf("Union = %+v", u)
	}
	// Zero rect is identity for Union.
	if got := (exp.Rect{}).Union(inner); got != inner {
		t.Errorf("Union with zero = %+v", got)
	}
	if got := inner.Union(exp.Rect{}); got != inner {
		t.Errorf("Union onto zero = %+v", got)
	}

	e := exp.R(10, 10, 20, 20).Expand(5)
	if e != exp.R(5, 5, 25, 25) {
		t.Errorf("Expand = %+v", e)
	}

	g := r.Geometry()
	if g.X0 != 0 || g.Y0 != 0 || g.X1 != 10 || g.Y1 != 20 {
		t.Errorf("Geometry = %+v", g)
	}
}

// --- Word / Words methods -------------------------------------------------

func TestWordAccessorsAndWordsMethods(t *testing.T) {
	d := openSample(t)
	p := page0(t, d)

	words, err := p.Words()
	if err != nil {
		t.Fatal(err)
	}
	if len(words) == 0 {
		t.Fatal("no words")
	}

	w := words[0]
	if w.Top() != w.Rect.Y0 || w.Bottom() != w.Rect.Y1 {
		t.Error("Top/Bottom mismatch")
	}
	if w.Left() != w.Rect.X0 || w.Right() != w.Rect.X1 {
		t.Error("Left/Right mismatch")
	}
	if w.Height() != w.Rect.Height() {
		t.Error("Height mismatch")
	}

	bounds := words.Bounds()
	if bounds.Empty() {
		t.Error("Bounds of all words should be non-empty")
	}
	if (exp.Words{}).Bounds() != (exp.Rect{}) {
		t.Error("Bounds of empty Words should be zero rect")
	}

	// LeftOf / RightOf are complementary about a split x.
	splitX := bounds.CenterX()
	left := words.LeftOf(splitX)
	right := words.RightOf(splitX)
	if len(left)+len(right) != len(words) {
		t.Errorf("LeftOf+RightOf = %d, want %d", len(left)+len(right), len(words))
	}

	// SortReading yields reading order (top-to-bottom, then left-to-right).
	sorted := words.SortReading()
	if len(sorted) != len(words) {
		t.Errorf("SortReading dropped words: %d vs %d", len(sorted), len(words))
	}

	lefts := words.Lefts()
	tops := words.Tops()
	if len(lefts) != len(words) || len(tops) != len(words) {
		t.Fatal("Lefts/Tops length mismatch")
	}
	if lefts[0] != words[0].Rect.X0 || tops[0] != words[0].Rect.Y0 {
		t.Error("Lefts/Tops values wrong")
	}

	// DropOutliers with explicit and defaulted factor keeps all normal text.
	if got := words.DropOutliers(2.2); len(got) > len(words) {
		t.Error("DropOutliers grew the slice")
	}
	if got := words.DropOutliers(0); len(got) == 0 {
		t.Error("DropOutliers(0) dropped everything")
	}
	if got := (exp.Words{}).DropOutliers(2); len(got) != 0 {
		t.Error("DropOutliers on empty should be empty")
	}
}

// --- Row methods + RowTolerance ------------------------------------------

func TestRowMethodsAndTolerance(t *testing.T) {
	d := openSample(t)
	p := page0(t, d)

	rows, err := p.Rows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no rows")
	}

	var nobleRow exp.Row
	for _, r := range rows {
		if strings.Contains(r.Text(), "Noble gases") {
			nobleRow = r
			break
		}
	}
	if len(nobleRow.Words) == 0 {
		t.Fatal("Noble gases row not found")
	}

	b := nobleRow.Bounds()
	if b.Empty() {
		t.Error("row Bounds empty")
	}
	// Band over the full row width returns the row's words.
	band := nobleRow.Band(b.X0-1, b.X1+1)
	if len(band) != len(nobleRow.Words) {
		t.Errorf("Band over full width = %d words, want %d", len(band), len(nobleRow.Words))
	}
	// A zero-width band off to the side returns nothing.
	if got := nobleRow.Band(b.X1+100, b.X1+200); len(got) != 0 {
		t.Errorf("off-band returned %d words", len(got))
	}

	// A tiny tolerance produces at least as many rows as the default.
	tight, err := p.Rows(exp.RowTolerance(0.1))
	if err != nil {
		t.Fatal(err)
	}
	if len(tight) < len(rows) {
		t.Errorf("tighter tolerance gave fewer rows (%d < %d)", len(tight), len(rows))
	}
}

// --- Page Search / WordsIn / CollectBlock --------------------------------

func TestSearchWordsInCollectBlock(t *testing.T) {
	d := openSample(t)
	p := page0(t, d)

	hits, err := p.Search("Noble")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("literal Search found nothing")
	}
	for _, h := range hits {
		if h.Rect.Empty() {
			t.Errorf("Search hit %q has empty rect", h.Text)
		}
	}

	b, err := p.Bound()
	if err != nil {
		t.Fatal(err)
	}
	in, err := p.WordsIn(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) == 0 {
		t.Fatal("WordsIn(full page) empty")
	}

	// CollectBlock down a known region. Find the Noble gases row and collect
	// the band covering its label column.
	rows, err := p.Rows()
	if err != nil {
		t.Fatal(err)
	}
	startIdx := -1
	var lo, hi float64
	for i, r := range rows {
		if strings.Contains(r.Text(), "Noble gases") {
			startIdx = i
			rb := r.Bounds()
			lo, hi = rb.X0-2, rb.X1+2
			break
		}
	}
	if startIdx < 0 {
		t.Fatal("could not anchor CollectBlock")
	}
	block := exp.CollectBlock(rows, startIdx, exp.BlockOptions{
		Lo: lo, Hi: hi, MaxLines: 2,
	})
	if len(block) == 0 {
		t.Fatal("CollectBlock returned nothing")
	}
	if !strings.Contains(block[0], "Noble") {
		t.Errorf("CollectBlock first line = %q, want it to contain Noble", block[0])
	}

	// Stop keyword: collecting with a Stop matching the very first band yields nothing.
	stopped := exp.CollectBlock(rows, startIdx, exp.BlockOptions{
		Lo: lo, Hi: hi, Stop: regexp.MustCompile("Noble"), GuardLeft: true,
	})
	if len(stopped) != 0 {
		t.Errorf("CollectBlock with Stop=Noble returned %v", stopped)
	}

	// MaxGap: a band far off to the right is all empty lines, so it stops early.
	empty := exp.CollectBlock(rows, 0, exp.BlockOptions{Lo: b.X1 + 500, Hi: b.X1 + 600})
	if len(empty) != 0 {
		t.Errorf("off-page band collected %v", empty)
	}
}

// --- Page query options coverage -----------------------------------------

func TestPageQueryOptions(t *testing.T) {
	d := openSample(t)
	p := page0(t, d)

	// CaseSensitive: lowercase label should not match capitalized "Noble".
	if _, ok, err := p.ValueRightOf("noble gases", exp.CaseSensitive()); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Error("case-sensitive match should have failed for wrong case")
	}

	// MaxGap clamps the swept value: a generous gap returns the whole tail,
	// a tighter one cuts it short (or stops before the first far column).
	full, ok, err := p.ValueRightOf("Noble gases", exp.MaxGap(500))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ValueRightOf with MaxGap(500) not found")
	}
	clipped, _, err := p.ValueRightOf("Noble gases", exp.MaxGap(15))
	if err != nil {
		t.Fatal(err)
	}
	if len(clipped) > len(full) {
		t.Errorf("MaxGap(15) value %q longer than MaxGap(500) value %q", clipped, full)
	}

	// Pad option exercised through ValueBelow.
	if _, _, err := p.ValueBelow("min", exp.Pad(4)); err != nil {
		t.Fatal(err)
	}
}

// --- Doc-level fan-out wrappers -------------------------------------------

func TestDocFanOut(t *testing.T) {
	d := openSample(t)

	hits, err := d.Find(`[A-Za-z]{3,}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("Doc.Find found nothing")
	}

	v, ok, err := d.ValueRightOf("Noble gases")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || v != "-269 -62 -170.5" {
		t.Errorf("Doc.ValueRightOf = %q ok=%v", v, ok)
	}

	if _, _, err := d.ValueBelow("min"); err != nil {
		t.Fatal(err)
	}
	// Missing label on all pages returns ok=false.
	if _, ok, _ := d.ValueRightOf("Definitely Not Present"); ok {
		t.Error("Doc.ValueRightOf found a nonexistent label")
	}
	if _, ok, _ := d.ValueBelow("Definitely Not Present"); ok {
		t.Error("Doc.ValueBelow found a nonexistent label")
	}
}

// --- Info / structure -----------------------------------------------------

func TestStructureAndText(t *testing.T) {
	d := openSample(t)
	p := page0(t, d)

	if _, err := d.Outline(); err != nil {
		t.Errorf("Outline: %v", err)
	}
	if _, err := p.Links(); err != nil {
		t.Errorf("Links: %v", err)
	}

	docText, err := d.Text()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(docText) == "" {
		t.Error("Doc.Text empty")
	}
	docLines, err := d.Lines()
	if err != nil {
		t.Fatal(err)
	}
	if len(docLines) == 0 {
		t.Error("Doc.Lines empty")
	}

	if d.Raw() == nil {
		t.Error("Doc.Raw nil")
	}
	if p.Raw() == nil {
		t.Error("Page.Raw nil")
	}

	pageText, err := p.Text()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(pageText) == "" {
		t.Error("Page.Text empty")
	}
	pageLines, err := p.Lines()
	if err != nil {
		t.Fatal(err)
	}
	if len(pageLines) == 0 {
		t.Error("Page.Lines empty")
	}

	b, err := p.Bound()
	if err != nil {
		t.Fatal(err)
	}
	if b.Empty() {
		t.Error("Page.Bound empty")
	}
}

// --- Tables: Doc-level, TablesIn, strategy options ------------------------

func TestTablesFanOutAndOptions(t *testing.T) {
	d := openSample(t)
	p := page0(t, d)

	dtabs, err := d.Tables()
	if err != nil {
		t.Fatal(err)
	}
	if len(dtabs) == 0 {
		t.Fatal("Doc.Tables found none")
	}

	ttabs, err := p.Tables(exp.TableText())
	if err != nil {
		t.Fatal(err)
	}
	if len(ttabs) == 0 {
		t.Fatal("TableText strategy found no tables")
	}
	region := ttabs[0].Region()
	if region.Empty() {
		t.Fatal("table region empty")
	}

	// TablesIn over the table's own region should return it.
	inside, err := p.TablesIn(region, exp.TableText())
	if err != nil {
		t.Fatal(err)
	}
	if len(inside) == 0 {
		t.Error("TablesIn over table region returned none")
	}
	// A disjoint region returns none.
	far, err := p.TablesIn(exp.R(region.X1+1000, region.Y1+1000, region.X1+1100, region.Y1+1100))
	if err != nil {
		t.Fatal(err)
	}
	if len(far) != 0 {
		t.Errorf("TablesIn over far region returned %d", len(far))
	}

	// TableLines strategy exercised (may legitimately find nothing).
	if _, err := p.Tables(exp.TableLines()); err != nil {
		t.Fatal(err)
	}
}

// --- Render: Image / SavePNG / SavePNGs / Thumbnail + options -------------

func TestRender(t *testing.T) {
	d := openSample(t)
	p := page0(t, d)

	img, err := p.Image(exp.Zoom(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		t.Error("rendered image has zero dimensions")
	}

	if _, err := p.Image(exp.Grayscale(), exp.DPI(36)); err != nil {
		t.Errorf("grayscale render: %v", err)
	}

	pngPath := filepath.Join(t.TempDir(), "page.png")
	if err := p.SavePNG(pngPath, exp.DPI(72)); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(pngPath); err != nil || fi.Size() == 0 {
		t.Errorf("SavePNG produced no file: err=%v", err)
	}

	dir := t.TempDir()
	paths, err := d.SavePNGs(dir, exp.Zoom(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != d.NumPages() {
		t.Errorf("SavePNGs returned %d paths, want %d", len(paths), d.NumPages())
	}
	for _, pth := range paths {
		if _, err := os.Stat(pth); err != nil {
			t.Errorf("SavePNGs path missing: %s", pth)
		}
	}

	thumb, err := d.Thumbnail(exp.Zoom(0.3))
	if err != nil {
		t.Fatal(err)
	}
	if thumb.Bounds().Empty() {
		t.Error("thumbnail empty")
	}
}

// --- Write / merge --------------------------------------------------------

func TestWriteAppendPDFSaveBytes(t *testing.T) {
	doc, err := exp.NewDoc()
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	if err := doc.AppendPDF(sample); err != nil {
		t.Fatalf("AppendPDF: %v", err)
	}
	if doc.NumPages() < 1 {
		t.Fatalf("pages after AppendPDF = %d", doc.NumPages())
	}

	b, err := doc.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("Bytes empty")
	}

	outPath := filepath.Join(t.TempDir(), "out.pdf")
	if err := doc.Save(outPath); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(outPath); err != nil || fi.Size() == 0 {
		t.Errorf("Save produced no file: err=%v", err)
	}

	// Reopen to confirm it is a valid PDF.
	reopened, err := exp.Open(outPath)
	if err != nil {
		t.Fatalf("reopen saved: %v", err)
	}
	reopened.Close()
}

// --- Images: per-image Save + Doc.SaveImages ------------------------------

func TestSaveImages(t *testing.T) {
	d, err := exp.Open(imgSample)
	if err != nil {
		t.Skipf("open %s: %v", imgSample, err)
	}
	defer d.Close()

	imgs, err := d.Images()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) == 0 {
		t.Skip("no embedded images")
	}

	// Single image Save.
	single := filepath.Join(t.TempDir(), "img."+imgs[0].Ext)
	if err := imgs[0].Save(single); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(single); err != nil || fi.Size() == 0 {
		t.Errorf("Image.Save produced no file: err=%v", err)
	}

	// Doc.SaveImages writes all of them.
	dir := t.TempDir()
	paths, err := d.SaveImages(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != len(imgs) {
		t.Errorf("SaveImages wrote %d, want %d", len(paths), len(imgs))
	}
	for _, pth := range paths {
		if _, err := os.Stat(pth); err != nil {
			t.Errorf("SaveImages path missing: %s", pth)
		}
	}
}

// --- ClusterFloats / Mean edge cases --------------------------------------

func TestClusterFloatsEdges(t *testing.T) {
	if got := exp.ClusterFloats(nil, 5); got != nil {
		t.Errorf("ClusterFloats(nil) = %v, want nil", got)
	}
	// Single tight cluster.
	got := exp.ClusterFloats([]float64{1, 2, 3, 4}, 5)
	if len(got) != 1 || len(got[0]) != 4 {
		t.Errorf("single cluster = %v", got)
	}
	if exp.Mean(nil) != 0 {
		t.Error("Mean(nil) should be 0")
	}
	if exp.Mean([]float64{2, 4}) != 3 {
		t.Error("Mean([2,4]) should be 3")
	}
}

// --- Error paths ----------------------------------------------------------

func TestOpenUnsupportedType(t *testing.T) {
	if _, err := exp.Open(12345); err == nil {
		t.Fatal("Open(int) should error")
	}
}

func TestOpenWrongPassword(t *testing.T) {
	// Build an encrypted PDF via the core API, then open it with a wrong
	// password and assert the ErrPassword sentinel.
	raw, err := gomupdf.Open(sample)
	if err != nil {
		t.Fatalf("core open: %v", err)
	}
	enc, err := raw.SaveEncryptedBytes("secret", "secret")
	raw.Close()
	if err != nil {
		t.Skipf("SaveEncryptedBytes unsupported: %v", err)
	}

	// Wrong password -> ErrPassword.
	if _, err := exp.Open(enc, exp.Password("wrong")); err == nil {
		t.Fatal("expected error for wrong password")
	} else if err != exp.ErrPassword {
		t.Errorf("got %v, want ErrPassword", err)
	}

	// Missing password -> ErrPassword.
	if _, err := exp.Open(enc); err != exp.ErrPassword {
		t.Errorf("missing-password got %v, want ErrPassword", err)
	}

	// Correct password -> success.
	ok, err := exp.Open(enc, exp.Password("secret"))
	if err != nil {
		t.Fatalf("correct password failed: %v", err)
	}
	ok.Close()
}
