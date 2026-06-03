package gomupdf

import (
	"errors"
	"fmt"
	"strings"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// DrawOptions styles a drawn primitive.
type DrawOptions struct {
	Stroke *[3]float64 // RGB 0..1 stroke color; nil → black
	Fill   *[3]float64 // RGB 0..1 fill color; nil → no fill
	Width  float64     // line width in points; ≤0 → 1
}

// lineWidth returns the effective line width.
func (o DrawOptions) lineWidth() float64 {
	if o.Width <= 0 {
		return 1
	}
	return o.Width
}

// paintOp returns "S", "f", or "B" based on whether Fill and Stroke are set.
func (o DrawOptions) paintOp() string {
	hasFill := o.Fill != nil
	// stroke is always applied (nil → black); but we follow the spec: if only
	// Fill is set and no Stroke color was supplied, still stroke in black (keep
	// simple per spec: always stroke).
	_ = hasFill
	if o.Fill != nil {
		return "B"
	}
	return "S"
}

// colorSetup builds the color/width preamble operators.
func (o DrawOptions) colorSetup() string {
	var b strings.Builder
	lw := o.lineWidth()
	fmt.Fprintf(&b, "%g w\n", lw)
	// stroke color
	if o.Stroke != nil {
		fmt.Fprintf(&b, "%g %g %g RG\n", o.Stroke[0], o.Stroke[1], o.Stroke[2])
	} else {
		b.WriteString("0 0 0 RG\n")
	}
	// fill color
	if o.Fill != nil {
		fmt.Fprintf(&b, "%g %g %g rg\n", o.Fill[0], o.Fill[1], o.Fill[2])
	}
	return b.String()
}

// appendContent delegates to the backend's generic content-stream writer.
func (p *Page) appendContent(fragment string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.drawContent(p.Number, fragment)
}

// appendTextContent delegates to the backend's text writer (ensures /F0 Helvetica).
func (p *Page) appendTextContent(fragment string) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.drawText(p.Number, fragment)
}

// DrawLine draws a line from a to b with the given options.
func (p *Page) DrawLine(a, b geometry.Point, opts DrawOptions) error {
	var frag strings.Builder
	frag.WriteString(opts.colorSetup())
	fmt.Fprintf(&frag, "%g %g m\n%g %g l\nS\n", a.X, a.Y, b.X, b.Y)
	return p.appendContent(frag.String())
}

// DrawRect draws a rectangle with the given options.
func (p *Page) DrawRect(r geometry.Rect, opts DrawOptions) error {
	var frag strings.Builder
	frag.WriteString(opts.colorSetup())
	// PDF re operator: x y width height re
	fmt.Fprintf(&frag, "%g %g %g %g re\n", r.X0, r.Y0, r.Width(), r.Height())
	frag.WriteString(opts.paintOp())
	frag.WriteString("\n")
	return p.appendContent(frag.String())
}

// DrawCircle approximates a circle with 4 cubic Bézier curves (kappa = 0.5523).
func (p *Page) DrawCircle(center geometry.Point, radius float64, opts DrawOptions) error {
	const kappa = 0.5523
	cx, cy, r := center.X, center.Y, radius
	k := kappa * r

	var frag strings.Builder
	frag.WriteString(opts.colorSetup())

	// Start at top (cx, cy+r), go clockwise: top→right→bottom→left→top
	fmt.Fprintf(&frag, "%g %g m\n", cx, cy+r)
	// top → right
	fmt.Fprintf(&frag, "%g %g %g %g %g %g c\n", cx+k, cy+r, cx+r, cy+k, cx+r, cy)
	// right → bottom
	fmt.Fprintf(&frag, "%g %g %g %g %g %g c\n", cx+r, cy-k, cx+k, cy-r, cx, cy-r)
	// bottom → left
	fmt.Fprintf(&frag, "%g %g %g %g %g %g c\n", cx-k, cy-r, cx-r, cy-k, cx-r, cy)
	// left → top
	fmt.Fprintf(&frag, "%g %g %g %g %g %g c\n", cx-r, cy+k, cx-k, cy+r, cx, cy+r)
	frag.WriteString("h\n")
	frag.WriteString(opts.paintOp())
	frag.WriteString("\n")
	return p.appendContent(frag.String())
}

// pdfEscape escapes (, ), and \ for use in a PDF string literal.
func pdfEscape(s string) string {
	var b strings.Builder
	for _, ch := range s {
		switch ch {
		case '(', ')', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// InsertTextbox lays out text inside rect with word wrapping (Helvetica, the
// given size), top-aligned, left-justified, and draws it into the page.
// Returns the number of lines that fit; text that overflows the rect is clipped.
func (p *Page) InsertTextbox(rect geometry.Rect, text string, size float64) (int, error) {
	// Measure using a separate Font (no page lock held during measuring).
	f, err := NewFont("Helvetica")
	if err != nil {
		return 0, fmt.Errorf("gomupdf: InsertTextbox: %w", err)
	}
	defer f.Close()

	maxWidth := rect.Width()
	lineHeight := size * 1.2
	spaceWidth := f.TextLength(" ", size)

	// Word-wrap: split into lines greedily.
	words := strings.Fields(text)
	var lines []string
	var currentLine strings.Builder
	currentWidth := 0.0
	firstWord := true

	for _, word := range words {
		ww := f.TextLength(word, size)
		if firstWord {
			currentLine.WriteString(word)
			currentWidth = ww
			firstWord = false
		} else if currentWidth+spaceWidth+ww <= maxWidth {
			currentLine.WriteByte(' ')
			currentLine.WriteString(word)
			currentWidth += spaceWidth + ww
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
			currentWidth = ww
		}
	}
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}
	if len(lines) == 0 {
		return 0, nil
	}

	// Determine how many lines fit vertically.
	// PDF coords: Y0 < Y1 (bottom < top). Start baseline near top (Y1 - size).
	// Each subsequent line baseline decreases by lineHeight.
	// A line fits if its baseline >= Y0 (i.e. it doesn't fall below the box bottom).
	x := rect.X0
	startBaseline := rect.Y1 - size
	lineCount := 0

	var frag strings.Builder
	frag.WriteString("BT\n/F0 ")
	fmt.Fprintf(&frag, "%g Tf\n", size)

	for i, line := range lines {
		baseline := startBaseline - float64(i)*lineHeight
		if baseline < rect.Y0 {
			break
		}
		fmt.Fprintf(&frag, "1 0 0 1 %g %g Tm\n(%s) Tj\n", x, baseline, pdfEscape(line))
		lineCount++
	}
	frag.WriteString("ET\n")

	if lineCount == 0 {
		return 0, nil
	}

	if err := p.appendTextContent(frag.String()); err != nil {
		return 0, err
	}
	return lineCount, nil
}
