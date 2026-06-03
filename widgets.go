package gomupdf

import (
	"errors"
	"strconv"
	"strings"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// Widget is a form field shown on a page.
type Widget struct {
	Index int    // 0-based position among the page's widgets
	Type  string // "text","checkbox","radiobutton","listbox","combobox","signature","button","unknown"
	Name  string // fully-qualified field name
	Value string // current field value
	Rect  geometry.Rect
}

// Widgets returns the form fields on the page in document order.
func (p *Page) Widgets() ([]Widget, error) {
	d := p.doc
	d.mu.Lock()
	if d.b == nil {
		d.mu.Unlock()
		return nil, errors.New("gomupdf: document closed")
	}
	raw, err := d.b.widgetsRaw(p.Number)
	d.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var out []Widget
	for i, ln := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if ln == "" {
			continue
		}
		// Format: typeName\tx0\ty0\tx1\ty1\tValue\tName
		parts := strings.SplitN(ln, "\t", 7)
		if len(parts) < 6 {
			continue
		}
		pf := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
		val := ""
		if len(parts) >= 7 {
			val = parts[5]
		}
		name := ""
		if len(parts) == 7 {
			name = parts[6]
		}
		out = append(out, Widget{
			Index: i,
			Type:  parts[0],
			Rect:  geometry.Rect{X0: pf(parts[1]), Y0: pf(parts[2]), X1: pf(parts[3]), Y1: pf(parts[4])},
			Value: val,
			Name:  name,
		})
	}
	return out, nil
}

// SetTextField sets the text value of the widget at index. Errors if it is not a text field.
func (p *Page) SetTextField(index int, value string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.setTextField(p.Number, index, value)
}

// SetCheckbox sets the checked state of the checkbox widget at index.
func (p *Page) SetCheckbox(index int, checked bool) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.setCheckbox(p.Number, index, checked)
}

// AddTextField creates a text form field named name over rect with an initial
// value, registering it in the document AcroForm. Effective on Save.
func (p *Page) AddTextField(name string, rect geometry.Rect, value string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.addTextField(p.Number, name, [4]float64{rect.X0, rect.Y0, rect.X1, rect.Y1}, value)
}
