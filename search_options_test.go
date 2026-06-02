package gomupdf

import (
	"strings"
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// needle is a word we know exists in small-table.pdf (from fixtures_test.go
// which asserts "Boiling Points" is in the table header row).
const searchNeedle = "Boiling"

func openSmallTablePage(t *testing.T) (*Document, *Page) {
	t.Helper()
	d := openFixture(t, "small-table.pdf")
	p, err := d.LoadPage(0)
	if err != nil {
		d.Close()
		t.Fatalf("LoadPage(0): %v", err)
	}
	return d, p
}

// TestSearchWithBaseline verifies that SearchWith with zero-value opts returns
// the same results as Search.
func TestSearchWithBaseline(t *testing.T) {
	d, p := openSmallTablePage(t)
	defer d.Close()

	base, err := p.Search(searchNeedle)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(base) == 0 {
		t.Fatalf("needle %q not found in fixture — pick a different needle", searchNeedle)
	}

	got, err := p.SearchWith(searchNeedle, SearchOptions{})
	if err != nil {
		t.Fatalf("SearchWith: %v", err)
	}
	if len(got) != len(base) {
		t.Errorf("zero-value opts: want %d hits, got %d", len(base), len(got))
	}
}

// TestSearchWithMaxHits verifies that MaxHits=1 returns exactly 1 result when
// the baseline has at least 1 hit.
func TestSearchWithMaxHits(t *testing.T) {
	d, p := openSmallTablePage(t)
	defer d.Close()

	base, _ := p.Search(searchNeedle)
	if len(base) == 0 {
		t.Skipf("needle %q not found in fixture", searchNeedle)
	}

	got, err := p.SearchWith(searchNeedle, SearchOptions{MaxHits: 1})
	if err != nil {
		t.Fatalf("SearchWith: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("MaxHits=1: want 1 hit, got %d", len(got))
	}
}

// TestSearchWithClipWholePageKeepsAll verifies that a clip rect covering the
// entire page keeps all baseline hits.
func TestSearchWithClipWholePageKeepsAll(t *testing.T) {
	d, p := openSmallTablePage(t)
	defer d.Close()

	base, _ := p.Search(searchNeedle)
	if len(base) == 0 {
		t.Skipf("needle %q not found in fixture", searchNeedle)
	}

	// A very large clip rect that encompasses any PDF page.
	fullPage := geometry.NewRect(0, 0, 10000, 10000)
	got, err := p.SearchWith(searchNeedle, SearchOptions{Clip: &fullPage})
	if err != nil {
		t.Fatalf("SearchWith: %v", err)
	}
	if len(got) != len(base) {
		t.Errorf("full-page clip: want %d hits, got %d", len(base), len(got))
	}
}

// TestSearchWithClipEmptyCornerReducesHits verifies that a tiny clip rect in
// an empty corner of the page returns fewer hits than the baseline.
func TestSearchWithClipEmptyCornerReducesHits(t *testing.T) {
	d, p := openSmallTablePage(t)
	defer d.Close()

	base, _ := p.Search(searchNeedle)
	if len(base) == 0 {
		t.Skipf("needle %q not found in fixture", searchNeedle)
	}

	// 1×1 pixel at the very top-left corner — extremely unlikely to overlap
	// any search hit quad in the fixture.
	corner := geometry.NewRect(0, 0, 1, 1)
	got, err := p.SearchWith(searchNeedle, SearchOptions{Clip: &corner})
	if err != nil {
		t.Fatalf("SearchWith: %v", err)
	}
	if len(got) >= len(base) {
		t.Errorf("tiny corner clip: expected fewer than %d hits, got %d", len(base), len(got))
	}
}

// TestSearchWithNoMatches verifies that a needle absent from the page returns
// an empty (or nil) slice with no error.
func TestSearchWithNoMatches(t *testing.T) {
	d, p := openSmallTablePage(t)
	defer d.Close()

	got, err := p.SearchWith("zzznomatchzzz", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchWith no-match: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("no-match needle: want 0 hits, got %d", len(got))
	}
}

// TestSearchWithClosedDocPropagatesError verifies that SearchWith propagates
// the "document closed" error from the underlying Search call.
func TestSearchWithClosedDocPropagatesError(t *testing.T) {
	d, p := openSmallTablePage(t)
	d.Close() // close before searching

	_, err := p.SearchWith(searchNeedle, SearchOptions{})
	if err == nil {
		t.Fatal("expected error on closed document, got nil")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("expected 'closed' in error, got: %v", err)
	}
}
