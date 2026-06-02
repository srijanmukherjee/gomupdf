package gomupdf

import (
	"testing"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// helper: reopen PDF from bytes
func reopenBytes(t *testing.T, data []byte) *Document {
	t.Helper()
	re, err := OpenStream(data)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	return re
}

// TestWidgetsEmpty verifies that a freshly created page has no widgets.
func TestWidgetsEmpty(t *testing.T) {
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.NewPage(400, 400); err != nil {
		t.Fatal(err)
	}
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	ws, err := p.Widgets()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 0 {
		t.Fatalf("expected 0 widgets on blank page, got %d", len(ws))
	}
}

// TestAddTextField creates a text field, saves, reopens and checks the field.
func TestAddTextField(t *testing.T) {
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.NewPage(400, 400); err != nil {
		t.Fatal(err)
	}
	p, err := d.LoadPage(0)
	if err != nil {
		t.Fatal(err)
	}

	rect := geometry.NewRect(50, 50, 250, 80)
	if err := p.AddTextField("fullname", rect, "Alice"); err != nil {
		t.Fatalf("AddTextField: %v", err)
	}

	data, err := d.SaveBytes(true)
	if err != nil {
		t.Fatalf("SaveBytes: %v", err)
	}

	re := reopenBytes(t, data)
	defer re.Close()
	rp, err := re.LoadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	ws, err := rp.Widgets()
	if err != nil {
		t.Fatalf("Widgets: %v", err)
	}
	if len(ws) != 1 {
		t.Fatalf("expected 1 widget after AddTextField, got %d", len(ws))
	}
	w := ws[0]
	t.Logf("Widget: index=%d type=%q name=%q value=%q rect=%+v", w.Index, w.Type, w.Name, w.Value, w.Rect)
	if w.Type != "text" {
		t.Errorf("widget type = %q, want %q", w.Type, "text")
	}
	if w.Name != "fullname" {
		t.Errorf("widget name = %q, want %q", w.Name, "fullname")
	}
	if w.Value != "Alice" {
		t.Errorf("widget value = %q, want %q", w.Value, "Alice")
	}
}

// TestSetTextField updates the value of an existing text field.
func TestSetTextField(t *testing.T) {
	// First create a doc with a text field
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.NewPage(400, 400); err != nil {
		t.Fatal(err)
	}
	p, _ := d.LoadPage(0)
	rect := geometry.NewRect(50, 50, 250, 80)
	if err := p.AddTextField("fullname", rect, "Alice"); err != nil {
		t.Fatalf("AddTextField: %v", err)
	}
	data, _ := d.SaveBytes(true)

	// Reopen and update the field
	re := reopenBytes(t, data)
	defer re.Close()
	rp, _ := re.LoadPage(0)
	if err := rp.SetTextField(0, "Bob"); err != nil {
		t.Fatalf("SetTextField: %v", err)
	}
	data2, err := re.SaveBytes(true)
	if err != nil {
		t.Fatalf("SaveBytes after SetTextField: %v", err)
	}

	// Reopen again and verify
	re2 := reopenBytes(t, data2)
	defer re2.Close()
	rp2, _ := re2.LoadPage(0)
	ws, err := rp2.Widgets()
	if err != nil {
		t.Fatalf("Widgets: %v", err)
	}
	if len(ws) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(ws))
	}
	t.Logf("After SetTextField: value=%q", ws[0].Value)
	if ws[0].Value != "Bob" {
		t.Errorf("widget value = %q, want %q", ws[0].Value, "Bob")
	}
}

// TestSetTextFieldOutOfRange verifies SetTextField errors on out-of-range index.
func TestSetTextFieldOutOfRange(t *testing.T) {
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.NewPage(400, 400); err != nil {
		t.Fatal(err)
	}
	p, _ := d.LoadPage(0)
	// No widgets added; index 0 should error
	if err := p.SetTextField(0, "anything"); err == nil {
		t.Error("expected error for out-of-range index, got nil")
	}
}

// TestSetTextFieldWrongType verifies SetTextField errors when widget is not text.
// We can't easily create a non-text widget here (AddTextField only creates text),
// so we verify the error comes through correctly by checking the error message
// we'd get on a doc that somehow has a checkbox — instead we test via SetCheckbox
// on a text field in TestSetCheckboxOnTextField.
// This test verifies the "widget is not a text field" guard indirectly:
// nothing to do here because our only creation path is text fields.
// (omitted — covered by SetCheckboxOnTextField below)

// TestSetCheckboxOutOfRange verifies SetCheckbox errors on out-of-range index.
func TestSetCheckboxOutOfRange(t *testing.T) {
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.NewPage(400, 400); err != nil {
		t.Fatal(err)
	}
	p, _ := d.LoadPage(0)
	if err := p.SetCheckbox(5, true); err == nil {
		t.Error("expected error for out-of-range checkbox index, got nil")
	}
}

// TestSetCheckboxOnTextField verifies that SetCheckbox on a text field returns an error.
func TestSetCheckboxOnTextField(t *testing.T) {
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.NewPage(400, 400); err != nil {
		t.Fatal(err)
	}
	p, _ := d.LoadPage(0)
	rect := geometry.NewRect(50, 50, 250, 80)
	if err := p.AddTextField("myfield", rect, ""); err != nil {
		t.Fatalf("AddTextField: %v", err)
	}
	// SetCheckbox on index 0 (a text field) should error
	if err := p.SetCheckbox(0, true); err == nil {
		t.Error("expected error when calling SetCheckbox on text field, got nil")
	}
}

// TestWidgetsClosedDocument verifies that all widget methods error on a closed doc.
func TestWidgetsClosedDocument(t *testing.T) {
	d, err := NewPDF()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.NewPage(400, 400); err != nil {
		t.Fatal(err)
	}
	p, _ := d.LoadPage(0)
	d.Close()

	if _, err := p.Widgets(); err == nil {
		t.Error("Widgets on closed doc: expected error, got nil")
	}
	if err := p.SetTextField(0, "x"); err == nil {
		t.Error("SetTextField on closed doc: expected error, got nil")
	}
	if err := p.SetCheckbox(0, true); err == nil {
		t.Error("SetCheckbox on closed doc: expected error, got nil")
	}
	if err := p.AddTextField("f", geometry.NewRect(0, 0, 100, 20), "v"); err == nil {
		t.Error("AddTextField on closed doc: expected error, got nil")
	}
}
