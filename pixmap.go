package gomupdf

import (
	"errors"
	"image"
	"image/png"
	"os"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// PixmapOptions controls rendering.
type PixmapOptions struct {
	Zoom     float64        // scale factor (1.0 = 72 DPI); <=0 means 1.0
	Gray     bool           // render in grayscale instead of RGB
	Alpha    bool           // include an alpha channel
	DPI      float64        // if > 0, overrides Zoom (Zoom = DPI/72)
	CMYK     bool           // render in CMYK (4 components); takes precedence over Gray
	Clip     *geometry.Rect // if non-nil, render only this region (PDF points)
	NoAnnots bool           // if true, render page contents only (skip annotations)
}

// Pixmap is a rendered raster of a page. Samples is row-major,
// Stride bytes per row, N components per pixel (1 gray, 2 gray+α, 3 rgb, 4 rgba).
type Pixmap struct {
	Width   int
	Height  int
	N       int
	Stride  int
	Alpha   bool
	Samples []byte
}

// Pixel returns the N component bytes at (x, y).
func (pm *Pixmap) Pixel(x, y int) []byte {
	if x < 0 || y < 0 || x >= pm.Width || y >= pm.Height {
		return nil
	}
	off := y*pm.Stride + x*pm.N
	return pm.Samples[off : off+pm.N]
}

// Image converts the pixmap to a standard library image.
func (pm *Pixmap) Image() (image.Image, error) {
	switch pm.N {
	case 1: // gray
		img := image.NewGray(image.Rect(0, 0, pm.Width, pm.Height))
		for y := 0; y < pm.Height; y++ {
			copy(img.Pix[y*img.Stride:], pm.Samples[y*pm.Stride:y*pm.Stride+pm.Width])
		}
		return img, nil
	case 3, 4: // rgb / rgba
		img := image.NewNRGBA(image.Rect(0, 0, pm.Width, pm.Height))
		for y := 0; y < pm.Height; y++ {
			for x := 0; x < pm.Width; x++ {
				s := pm.Samples[y*pm.Stride+x*pm.N:]
				d := img.Pix[y*img.Stride+x*4:]
				d[0], d[1], d[2] = s[0], s[1], s[2]
				if pm.N == 4 {
					d[3] = s[3]
				} else {
					d[3] = 255
				}
			}
		}
		return img, nil
	case 2: // gray + alpha → expand to NRGBA
		img := image.NewNRGBA(image.Rect(0, 0, pm.Width, pm.Height))
		for y := 0; y < pm.Height; y++ {
			for x := 0; x < pm.Width; x++ {
				s := pm.Samples[y*pm.Stride+x*2:]
				d := img.Pix[y*img.Stride+x*4:]
				d[0], d[1], d[2], d[3] = s[0], s[0], s[0], s[1]
			}
		}
		return img, nil
	}
	return nil, errors.New("gomupdf: unsupported pixmap component count")
}

// PNG encodes the pixmap as PNG bytes.
func (pm *Pixmap) PNG() ([]byte, error) {
	img, err := pm.Image()
	if err != nil {
		return nil, err
	}
	var buf bytesBuffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.b, nil
}

// SavePNG writes the pixmap to a PNG file.
func (pm *Pixmap) SavePNG(path string) error {
	data, err := pm.PNG()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// tiny io.Writer to avoid importing bytes just for a buffer.
type bytesBuffer struct{ b []byte }

func (w *bytesBuffer) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}
