package gomupdf

import "testing"

// buildPages makes an n-page blank PDF.
func buildPages(t *testing.T, n int) *Document {
	t.Helper()
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if err := d.NewPage(200, 200); err != nil {
			d.Close()
			t.Fatal(err)
		}
	}
	return d
}

// Labels resolve correctly per page and survive a save/reopen round-trip:
// pages 0–1 are lowercase roman (i, ii), pages 2+ are decimal with an "A-"
// prefix starting at 1 (A-1, A-2, A-3).
func TestPageLabelsResolve(t *testing.T) {
	d := buildPages(t, 5)
	rules := []PageLabel{
		{StartPage: 0, Style: LabelRomanLower, Start: 1},
		{StartPage: 2, Style: LabelDecimal, Prefix: "A-", Start: 1},
	}
	if err := d.SetPageLabels(rules); err != nil {
		d.Close()
		t.Fatal(err)
	}
	data, err := d.SaveBytes(true)
	d.Close()
	if err != nil {
		t.Fatal(err)
	}

	d2, err := OpenStream(data)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()

	want := []string{"i", "ii", "A-1", "A-2", "A-3"}
	for i, w := range want {
		p, _ := d2.LoadPage(i)
		got, err := p.Label()
		if err != nil {
			t.Fatal(err)
		}
		if got != w {
			t.Errorf("page %d label = %q, want %q", i, got, w)
		}
	}
}

// The rules read back match what was written.
func TestPageLabelsReadBack(t *testing.T) {
	d := buildPages(t, 4)
	rules := []PageLabel{
		{StartPage: 0, Style: LabelRomanUpper, Start: 1},
		{StartPage: 1, Style: LabelDecimal, Prefix: "p", Start: 10},
	}
	if err := d.SetPageLabels(rules); err != nil {
		d.Close()
		t.Fatal(err)
	}
	data, _ := d.SaveBytes(true)
	d.Close()

	d2, _ := OpenStream(data)
	defer d2.Close()
	got, err := d2.PageLabels()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(rules) {
		t.Fatalf("read %d rules, want %d", len(got), len(rules))
	}
	for i, w := range rules {
		if got[i].StartPage != w.StartPage || got[i].Style != w.Style ||
			got[i].Prefix != w.Prefix || got[i].Start != w.Start {
			t.Errorf("rule %d = %+v, want %+v", i, got[i], w)
		}
	}
}

// Passing nil clears labels; Label then reports the empty string.
func TestPageLabelsClear(t *testing.T) {
	d := buildPages(t, 3)
	_ = d.SetPageLabels([]PageLabel{{StartPage: 0, Style: LabelDecimal, Start: 1}})
	if err := d.SetPageLabels(nil); err != nil {
		d.Close()
		t.Fatal(err)
	}
	data, _ := d.SaveBytes(true)
	d.Close()

	d2, _ := OpenStream(data)
	defer d2.Close()
	rules, _ := d2.PageLabels()
	if len(rules) != 0 {
		t.Errorf("expected no rules after clear, got %d", len(rules))
	}
	p, _ := d2.LoadPage(0)
	if lbl, _ := p.Label(); lbl != "" {
		t.Errorf("label after clear = %q, want empty", lbl)
	}
}

// A document with no labels resolves to empty strings, not an error.
func TestPageLabelsAbsent(t *testing.T) {
	d := buildPages(t, 2)
	defer d.Close()
	rules, err := d.PageLabels()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Errorf("expected no rules, got %d", len(rules))
	}
	p, _ := d.LoadPage(0)
	if lbl, _ := p.Label(); lbl != "" {
		t.Errorf("label = %q, want empty", lbl)
	}
}

// Operations on a closed document error cleanly.
func TestPageLabelsClosedDoc(t *testing.T) {
	d := buildPages(t, 1)
	d.Close()
	if err := d.SetPageLabels(nil); err == nil {
		t.Error("SetPageLabels on closed doc should error")
	}
	if _, err := d.PageLabels(); err == nil {
		t.Error("PageLabels on closed doc should error")
	}
}
