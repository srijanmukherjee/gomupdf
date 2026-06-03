package gomupdf

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// buildTextPages makes an n-page PDF where page i carries the unique text
// "PAGE-i", round-tripped through save/reopen so the text is extractable.
func buildTextPages(t testing.TB, n int) *Document {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if err := d.NewPage(300, 300); err != nil {
			d.Close()
			t.Fatal(err)
		}
		p, _ := d.LoadPage(i)
		if err := p.InsertText(50, 150, 24, fmt.Sprintf("PAGE-%d", i)); err != nil {
			d.Close()
			t.Fatalf("insert text page %d: %v", i, err)
		}
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	rd, err := OpenStream(data)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	return rd
}

// buildHeavyPages makes an n-page PDF whose pages carry hundreds of filled
// vector circles each, so rasterization is CPU-bound (the case parallel
// rendering actually accelerates), round-tripped through save/reopen.
func buildHeavyPages(t testing.TB, n int) *Document {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	fill := [3]float64{0.2, 0.4, 0.8}
	for i := 0; i < n; i++ {
		if err := d.NewPage(400, 400); err != nil {
			d.Close()
			t.Fatal(err)
		}
		p, _ := d.LoadPage(i)
		for j := 0; j < 300; j++ {
			cx := float64(20 + (j*37)%360)
			cy := float64(20 + (j*53)%360)
			if err := p.DrawCircle(geometry.Point{X: cx, Y: cy}, 40, DrawOptions{Fill: &fill, Width: 1}); err != nil {
				d.Close()
				t.Fatalf("draw circle: %v", err)
			}
		}
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	rd, err := OpenStream(data)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	return rd
}

func seq(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

// Concurrent text extraction matches serial extraction page-for-page, and the
// results come back in input order. Run under -race to catch context misuse.
func TestTextByPageConcurrent(t *testing.T) {
	const n = 24
	d := buildTextPages(t, n)
	defer d.Close()

	serial, err := d.TextByPage()
	if err != nil {
		t.Fatal(err)
	}
	conc, err := d.TextByPageConcurrent(context.Background(), WithWorkers(8))
	if err != nil {
		t.Fatalf("concurrent: %v", err)
	}
	if len(conc) != len(serial) {
		t.Fatalf("len = %d, want %d", len(conc), len(serial))
	}
	for i := range serial {
		if conc[i] != serial[i] {
			t.Errorf("page %d: concurrent %q != serial %q", i, conc[i], serial[i])
		}
		if !strings.Contains(conc[i], fmt.Sprintf("PAGE-%d", i)) {
			t.Errorf("page %d text %q missing marker", i, conc[i])
		}
	}
}

// MapPages preserves input order even when pages are processed out of order and
// the slice is not 0..N-1.
func TestMapPagesOrder(t *testing.T) {
	d := buildTextPages(t, 10)
	defer d.Close()

	pages := []int{9, 0, 5, 2, 7, 1}
	got, err := MapPages(context.Background(), d, pages, (*Page).GetText, WithWorkers(4))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(pages) {
		t.Fatalf("len = %d, want %d", len(got), len(pages))
	}
	for i, pno := range pages {
		want := fmt.Sprintf("PAGE-%d", pno)
		if !strings.Contains(got[i], want) {
			t.Errorf("result[%d] = %q, want it to contain %q", i, got[i], want)
		}
	}
}

// RenderPages produces one pixmap per page with sane dimensions, matching a
// serial render of the same page.
func TestRenderPagesConcurrent(t *testing.T) {
	const n = 12
	d := buildTextPages(t, n)
	defer d.Close()

	pms, err := d.RenderPages(context.Background(), seq(n), PixmapOptions{}, WithWorkers(6))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(pms) != n {
		t.Fatalf("len = %d, want %d", len(pms), n)
	}
	p0, _ := d.LoadPage(0)
	ref, err := p0.Pixmap()
	if err != nil {
		t.Fatal(err)
	}
	for i, pm := range pms {
		if pm == nil {
			t.Fatalf("page %d: nil pixmap", i)
		}
		if pm.Width != ref.Width || pm.Height != ref.Height {
			t.Errorf("page %d: %dx%d, want %dx%d", i, pm.Width, pm.Height, ref.Width, ref.Height)
		}
		if len(pm.Samples) == 0 {
			t.Errorf("page %d: empty samples", i)
		}
	}
}

// A canceled context short-circuits the operation.
func TestMapPagesCancel(t *testing.T) {
	d := buildTextPages(t, 16)
	defer d.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled before any work dispatches

	_, err := MapPages(ctx, d, seq(16), (*Page).GetText, WithWorkers(4))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// The first fn error is propagated and cancels the remaining work.
func TestMapPagesErrorPropagation(t *testing.T) {
	d := buildTextPages(t, 20)
	defer d.Close()

	sentinel := errors.New("boom")
	var calls int32
	_, err := MapPages(context.Background(), d, seq(20), func(p *Page) (string, error) {
		atomic.AddInt32(&calls, 1)
		if p.Number == 7 {
			return "", sentinel
		}
		return p.GetText()
	}, WithWorkers(4))
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

// Single-worker and over-provisioned worker counts both yield correct results.
func TestMapPagesWorkerCounts(t *testing.T) {
	const n = 8
	d := buildTextPages(t, n)
	defer d.Close()

	for _, w := range []int{1, 3, n, n * 4} {
		got, err := MapPages(context.Background(), d, seq(n), (*Page).GetText, WithWorkers(w))
		if err != nil {
			t.Fatalf("workers=%d: %v", w, err)
		}
		for i := 0; i < n; i++ {
			if !strings.Contains(got[i], fmt.Sprintf("PAGE-%d", i)) {
				t.Errorf("workers=%d page %d: %q", w, i, got[i])
			}
		}
	}
}

// Out-of-range page indices are rejected before any work runs.
func TestMapPagesOutOfRange(t *testing.T) {
	d := buildTextPages(t, 3)
	defer d.Close()

	if _, err := MapPages(context.Background(), d, []int{0, 5}, (*Page).GetText); err == nil {
		t.Fatal("expected out-of-range error")
	}
}

// Operating on a closed document is reported, not crashed.
func TestMapPagesClosed(t *testing.T) {
	d := buildTextPages(t, 3)
	d.Close()
	if _, err := MapPages(context.Background(), d, []int{0}, (*Page).GetText); err == nil {
		t.Fatal("expected closed-document error")
	}
}

// In-memory documents (no backing bytes) cannot spawn workers and fall back to
// serial execution, still returning correct results.
func TestMapPagesInMemoryFallback(t *testing.T) {
	d := buildPages(t, 4) // NewPDF-built, never saved/reopened: no data buffer
	defer d.Close()
	got, err := MapPages(context.Background(), d, seq(4), func(p *Page) (int, error) {
		return p.Number, nil
	}, WithWorkers(4))
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		if got[i] != i {
			t.Errorf("result[%d] = %d", i, got[i])
		}
	}
}

// Empty page set returns no results and no error.
func TestMapPagesEmpty(t *testing.T) {
	d := buildTextPages(t, 3)
	defer d.Close()
	got, err := MapPages(context.Background(), d, nil, (*Page).GetText)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

// BenchmarkRenderSerialVsConcurrent contrasts whole-document rendering serially
// against RenderPages across the default worker pool. Rendering is done at 300
// DPI so rasterization cost (the parallelizable work) dominates the fixed
// per-worker document-reopen overhead — i.e. the realistic case the concurrency
// feature targets. Run with -benchmem:
//
//	go test -bench BenchmarkRender -benchmem
//
// (On trivial/blank pages the per-worker reopen cost can exceed the render work
// and serial wins; the win shows up once pages carry real rendering cost.)
func BenchmarkRenderSerialVsConcurrent(b *testing.B) {
	const n = 64
	d := buildHeavyPages(b, n)
	defer d.Close()
	pages := seq(n)
	opts := PixmapOptions{DPI: 200}

	b.Run("serial", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, pno := range pages {
				p, _ := d.LoadPage(pno)
				if _, err := p.Pixmap(opts); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
	b.Run("concurrent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := d.RenderPages(context.Background(), pages, opts); err != nil {
				b.Fatal(err)
			}
		}
	})
}
