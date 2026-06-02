package gomupdf

import (
	"bytes"
	"errors"
	"fmt"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// colorComps returns the number of color (non-alpha) components for the pixmap.
// If Alpha is true, the last component is alpha, so color components = N-1.
// Otherwise all N components are color.
func (pm *Pixmap) colorComps() int {
	if pm.Alpha {
		return pm.N - 1
	}
	return pm.N
}

// isCMYK reports whether the pixmap is CMYK (N==4 and no alpha channel).
func (pm *Pixmap) isCMYK() bool {
	return pm.N == 4 && !pm.Alpha
}

// Invert inverts the color components (each → 255−v) in place; alpha is preserved.
func (pm *Pixmap) Invert() {
	cc := pm.colorComps()
	for y := 0; y < pm.Height; y++ {
		row := pm.Samples[y*pm.Stride:]
		for x := 0; x < pm.Width; x++ {
			px := row[x*pm.N:]
			for c := 0; c < cc; c++ {
				px[c] = 255 - px[c]
			}
		}
	}
}

// Gamma applies gamma correction to color components in place: v' = 255·(v/255)^(1/gamma).
// gamma ≤ 0 is a no-op. Alpha is preserved.
func (pm *Pixmap) Gamma(gamma float64) {
	if gamma <= 0 {
		return
	}
	// Build a lookup table for speed.
	var lut [256]byte
	inv := 1.0 / gamma
	for i := range lut {
		v := math.Pow(float64(i)/255.0, inv) * 255.0
		if v > 255 {
			v = 255
		}
		lut[i] = byte(math.Round(v))
	}

	cc := pm.colorComps()
	for y := 0; y < pm.Height; y++ {
		row := pm.Samples[y*pm.Stride:]
		for x := 0; x < pm.Width; x++ {
			px := row[x*pm.N:]
			for c := 0; c < cc; c++ {
				px[c] = lut[px[c]]
			}
		}
	}
}

// JPEG encodes the pixmap as JPEG (quality 1–100). Only gray/RGB pixmaps are
// supported; returns an error for CMYK.
func (pm *Pixmap) JPEG(quality int) ([]byte, error) {
	if pm.isCMYK() {
		return nil, errors.New("gomupdf: JPEG encoding not supported for CMYK pixmaps")
	}
	img, err := pm.Image()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PNM encodes the pixmap as binary PNM (P5 for gray, P6 for RGB). Alpha and
// CMYK are not representable; returns an error for CMYK.
func (pm *Pixmap) PNM() ([]byte, error) {
	if pm.isCMYK() {
		return nil, errors.New("gomupdf: PNM encoding not supported for CMYK pixmaps")
	}

	cc := pm.colorComps()
	var magic string
	switch cc {
	case 1:
		magic = "P5"
	case 3:
		magic = "P6"
	default:
		return nil, fmt.Errorf("gomupdf: PNM unsupported color component count: %d", cc)
	}

	header := fmt.Sprintf("%s\n%d %d\n255\n", magic, pm.Width, pm.Height)
	out := make([]byte, 0, len(header)+pm.Width*pm.Height*cc)
	out = append(out, []byte(header)...)

	for y := 0; y < pm.Height; y++ {
		row := pm.Samples[y*pm.Stride:]
		for x := 0; x < pm.Width; x++ {
			px := row[x*pm.N:]
			out = append(out, px[:cc]...)
		}
	}
	return out, nil
}

// Bytes encodes the pixmap in the named format: "png", "jpeg"/"jpg", or "pnm".
func (pm *Pixmap) Bytes(format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "png":
		return pm.PNG()
	case "jpeg", "jpg":
		return pm.JPEG(85)
	case "pnm":
		return pm.PNM()
	default:
		return nil, fmt.Errorf("gomupdf: unknown format %q", format)
	}
}

// Save writes the pixmap to path, choosing the format from the file extension
// (.png, .jpg/.jpeg, .pnm). Unknown extensions return an error.
func (pm *Pixmap) Save(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	var data []byte
	var err error
	switch ext {
	case ".png":
		data, err = pm.PNG()
	case ".jpg", ".jpeg":
		data, err = pm.JPEG(85)
	case ".pnm":
		data, err = pm.PNM()
	default:
		return fmt.Errorf("gomupdf: unknown file extension %q", ext)
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
