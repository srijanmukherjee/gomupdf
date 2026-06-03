package gomupdf

import (
	"errors"
	"strconv"
	"strings"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// Page geometry: rotation and the /MediaBox and /CropBox page boxes.
//
// PDF defines several page boxes; gomupdf exposes the two that matter for
// rendering and layout: the MediaBox (the full physical page) and the CropBox
// (the visible region, defaulting to the MediaBox when absent). All boxes are
// reported in unrotated PDF points (origin bottom-left); use Rotation to read
// the display rotation separately. These mirror PyMuPDF's page.rotation /
// page.mediabox / page.cropbox.

// pageGeometry reads rotation, MediaBox, and CropBox in one backend round-trip.
func (p *Page) pageGeometry() (rot int, media, crop geometry.Rect, hasCrop bool, err error) {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return 0, geometry.Rect{}, geometry.Rect{}, false, errors.New("gomupdf: document closed")
	}
	raw, err := d.b.geometryRaw(p.Number)
	if err != nil {
		return 0, geometry.Rect{}, geometry.Rect{}, false, err
	}
	f := strings.Fields(raw)
	if len(f) != 10 {
		return 0, geometry.Rect{}, geometry.Rect{}, false, errors.New("gomupdf: bad page geometry output")
	}
	rot, _ = strconv.Atoi(f[0])
	pf := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	media = geometry.Rect{X0: pf(f[1]), Y0: pf(f[2]), X1: pf(f[3]), Y1: pf(f[4])}
	crop = geometry.Rect{X0: pf(f[5]), Y0: pf(f[6]), X1: pf(f[7]), Y1: pf(f[8])}
	hasCrop = f[9] == "1"
	return rot, media, crop, hasCrop, nil
}

// normalizeAngle reduces deg into [0, 360).
func normalizeAngle(deg int) int {
	deg %= 360
	if deg < 0 {
		deg += 360
	}
	return deg
}

// Rotation returns the page's display rotation in degrees: one of 0, 90, 180,
// or 270. The value is normalized into [0, 360).
func (p *Page) Rotation() (int, error) {
	rot, _, _, _, err := p.pageGeometry()
	if err != nil {
		return 0, err
	}
	return normalizeAngle(rot), nil
}

// SetRotation sets the page's display rotation. deg must be a multiple of 90;
// it is normalized into [0, 360). Changes take effect on the next Save.
func (p *Page) SetRotation(deg int) error {
	if deg%90 != 0 {
		return errors.New("gomupdf: rotation must be a multiple of 90")
	}
	deg = normalizeAngle(deg)
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.setRotation(p.Number, deg)
}

// MediaBox returns the page's /MediaBox — the full physical page rectangle in
// unrotated PDF points.
func (p *Page) MediaBox() (geometry.Rect, error) {
	_, media, _, _, err := p.pageGeometry()
	return media, err
}

// CropBox returns the page's /CropBox — the visible region in unrotated PDF
// points. When the page declares no CropBox, the MediaBox is returned.
func (p *Page) CropBox() (geometry.Rect, error) {
	_, _, crop, _, err := p.pageGeometry()
	return crop, err
}

// setBox is the shared entry point for SetMediaBox/SetCropBox.
func (p *Page) setBox(which int, r geometry.Rect) error {
	d := p.doc
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.b == nil {
		return errors.New("gomupdf: document closed")
	}
	return d.b.setBox(p.Number, which, r.X0, r.Y0, r.X1, r.Y1)
}

// SetMediaBox sets the page's /MediaBox. The rectangle is given in unrotated
// PDF points. Changes take effect on the next Save.
func (p *Page) SetMediaBox(r geometry.Rect) error { return p.setBox(0, r) }

// SetCropBox sets the page's /CropBox. The rectangle is given in unrotated PDF
// points and is clamped to the MediaBox by MuPDF. Changes take effect on the
// next Save.
func (p *Page) SetCropBox(r geometry.Rect) error { return p.setBox(1, r) }
