package gomupdf

import (
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// TestAnnotationsEmptyPage verifies that a page with no annotations returns
// an empty slice without error.
func TestAnnotationsEmptyPage(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("load page: %v", err)
	}
	annots, err := p.Annotations()
	if err != nil {
		t.Fatalf("Annotations: %v", err)
	}
	if len(annots) != 0 {
		t.Errorf("expected 0 annotations on fresh page, got %d", len(annots))
	}
}

// TestAnnotationsClosedDoc verifies that Annotations returns an error on a
// closed document.
func TestAnnotationsClosedDoc(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	d.Close()
	p := &Page{doc: d, Number: 0}
	_, err := p.Annotations()
	if err == nil {
		t.Fatal("expected error from closed document, got nil")
	}
}

// TestDeleteAnnotationClosedDoc verifies DeleteAnnotation returns error on closed doc.
func TestDeleteAnnotationClosedDoc(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	d.Close()
	p := &Page{doc: d, Number: 0}
	err := p.DeleteAnnotation(0)
	if err == nil {
		t.Fatal("expected error from closed document, got nil")
	}
}

// TestAddHighlightAndList adds a highlight annotation and checks it appears
// in Annotations() with the right Type and a Rect that overlaps the search hit.
func TestAddHighlightAndList(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("load page: %v", err)
	}

	quads, err := p.Search("Boiling")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(quads) == 0 {
		t.Fatal("Search for 'Boiling' returned no quads")
	}

	if err := p.AddHighlight(quads); err != nil {
		t.Fatalf("AddHighlight: %v", err)
	}

	annots, err := p.Annotations()
	if err != nil {
		t.Fatalf("Annotations: %v", err)
	}
	if len(annots) == 0 {
		t.Fatal("expected at least one annotation after AddHighlight, got 0")
	}

	var found *Annotation
	for i := range annots {
		if annots[i].Type == "Highlight" {
			found = &annots[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("no annotation with Type==\"Highlight\"; got %v", annots)
	}

	// The highlight's Rect should overlap the search hit quad's bounding rect.
	hitRect := quads[0].Rect()
	if !found.Rect.Intersects(hitRect) {
		t.Errorf("highlight Rect %v does not intersect search hit %v", found.Rect, hitRect)
	}
}

// TestAddHighlightPersists adds a highlight, saves, reopens, and verifies it persists.
func TestAddHighlightPersists(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("load page: %v", err)
	}

	quads, err := p.Search("Boiling")
	if err != nil || len(quads) == 0 {
		t.Fatalf("Search: %v (quads=%d)", err, len(quads))
	}
	if err := p.AddHighlight(quads); err != nil {
		t.Fatalf("AddHighlight: %v", err)
	}

	data, err := d.SaveBytes(true)
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}

	d2, err := OpenStream(data)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer d2.Close()

	p2, err := d2.LoadPage(0)
	if err != nil {
		t.Fatalf("load page from reopened doc: %v", err)
	}
	annots, err := p2.Annotations()
	if err != nil {
		t.Fatalf("Annotations from reopened doc: %v", err)
	}
	found := false
	for _, a := range annots {
		if a.Type == "Highlight" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Highlight annotation did not persist after save/reopen; got %v", annots)
	}
}

// TestAddAllMarkupTypes verifies AddUnderline, AddStrikeout, and AddSquiggly
// each produce an annotation of the expected type.
func TestAddAllMarkupTypes(t *testing.T) {
	type tc struct {
		name     string
		addFn    func(*Page, []geometry.Quad) error
		wantType string
	}
	cases := []tc{
		{"Underline", (*Page).AddUnderline, "Underline"},
		{"Strikeout", (*Page).AddStrikeout, "StrikeOut"},
		{"Squiggly", (*Page).AddSquiggly, "Squiggly"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := openFixture(t, "small-table.pdf")
			defer d.Close()
			p, err := d.LoadPage(0)
			if err != nil {
				t.Fatalf("load page: %v", err)
			}
			quads, err := p.Search("Boiling")
			if err != nil || len(quads) == 0 {
				t.Fatalf("Search: %v (quads=%d)", err, len(quads))
			}
			if err := c.addFn(p, quads); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			annots, err := p.Annotations()
			if err != nil {
				t.Fatalf("Annotations: %v", err)
			}
			found := false
			for _, a := range annots {
				t.Logf("annotation type: %q", a.Type)
				if a.Type == c.wantType {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected annotation with Type==%q, got %v", c.wantType, annots)
			}
		})
	}
}

// TestDeleteAnnotationReducesCount adds two annotations and verifies that
// DeleteAnnotation(0) reduces the count by one.
func TestDeleteAnnotationReducesCount(t *testing.T) {
	d := openFixture(t, "small-table.pdf")
	defer d.Close()
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatalf("load page: %v", err)
	}

	quads, err := p.Search("Boiling")
	if err != nil || len(quads) == 0 {
		t.Fatalf("Search: %v (quads=%d)", err, len(quads))
	}
	if err := p.AddHighlight(quads); err != nil {
		t.Fatalf("AddHighlight: %v", err)
	}
	if err := p.AddUnderline(quads); err != nil {
		t.Fatalf("AddUnderline: %v", err)
	}

	before, err := p.Annotations()
	if err != nil {
		t.Fatalf("Annotations before delete: %v", err)
	}
	if len(before) < 2 {
		t.Fatalf("expected at least 2 annotations, got %d", len(before))
	}

	if err := p.DeleteAnnotation(0); err != nil {
		t.Fatalf("DeleteAnnotation: %v", err)
	}

	after, err := p.Annotations()
	if err != nil {
		t.Fatalf("Annotations after delete: %v", err)
	}
	if len(after) != len(before)-1 {
		t.Errorf("expected %d annotations after delete, got %d", len(before)-1, len(after))
	}
}
