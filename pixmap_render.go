package gomupdf

import (
	"encoding/binary"
	"errors"
)

// parseBlob converts the 5×int32 header + samples blob into a Pixmap.
func parseBlob(blob []byte) (*Pixmap, error) {
	if len(blob) < 20 {
		return nil, errors.New("gomupdf: short pixmap blob")
	}
	rd := func(i int) int { return int(int32(binary.LittleEndian.Uint32(blob[i*4 : i*4+4]))) }
	// make a Go-owned copy of the samples so the caller owns the memory cleanly
	samples := make([]byte, len(blob)-20)
	copy(samples, blob[20:])
	return &Pixmap{
		Width:   rd(0),
		Height:  rd(1),
		N:       rd(2),
		Stride:  rd(3),
		Alpha:   rd(4) != 0,
		Samples: samples,
	}, nil
}

// Pixmap renders the page with the given options.
// Resolution precedence: DPI>0 → zoom=DPI/72; else Zoom (≤0 means 1).
// Colorspace precedence: CMYK → device CMYK; else Gray → device gray; else RGB.
func (p *Page) Pixmap(opts ...PixmapOptions) (*Pixmap, error) {
	o := PixmapOptions{Zoom: 1}
	if len(opts) > 0 {
		o = opts[0]
	}

	d := p.doc
	d.mu.Lock()
	if d.b == nil {
		d.mu.Unlock()
		return nil, errors.New("gomupdf: document closed")
	}
	blob, err := d.b.pixmap(p.Number, o)
	d.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return parseBlob(blob)
}
