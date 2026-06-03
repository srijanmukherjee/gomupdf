package gomupdf

import (
	"runtime"
	"sync"
)

// Font is a loaded font usable for measuring text. Backed by the active engine;
// call Close when done (a finalizer is a backstop).
type Font struct {
	mu sync.Mutex
	b  fontBackend
}

// NewFont loads one of the 14 standard PDF fonts by name (e.g. "Helvetica",
// "Times-Roman", "Courier", "Helvetica-Bold", "Symbol", "ZapfDingbats").
func NewFont(name string) (*Font, error) {
	if defaultDriver == nil {
		return nil, errNoBackend
	}
	b, err := defaultDriver.newFont(name)
	if err != nil {
		return nil, err
	}
	f := &Font{b: b}
	runtime.SetFinalizer(f, (*Font).Close)
	return f, nil
}

// TextLength returns the rendered width of text at the given font size (points).
func (f *Font) TextLength(text string, size float64) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.b == nil {
		return 0
	}
	var total float64
	for _, r := range text {
		total += f.b.advance(r)
	}
	return total * size
}

// Advance returns the horizontal advance of a single rune at the given size.
func (f *Font) Advance(r rune, size float64) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.b == nil {
		return 0
	}
	return f.b.advance(r) * size
}

// Ascender returns the font's ascender as a fraction of the em (×size for points).
func (f *Font) Ascender() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.b == nil {
		return 0
	}
	return f.b.ascender()
}

// Descender returns the font's descender as a fraction of the em (negative).
func (f *Font) Descender() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.b == nil {
		return 0
	}
	return f.b.descender()
}

// Name returns the font's name.
func (f *Font) Name() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.b == nil {
		return ""
	}
	return f.b.fontName()
}

// Close releases the underlying font. Safe to call twice.
func (f *Font) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.b != nil {
		f.b.close()
		f.b = nil
	}
	runtime.SetFinalizer(f, nil)
}
