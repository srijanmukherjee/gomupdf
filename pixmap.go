package gomupdf

import (
	"encoding/binary"
	"errors"
	"image"
	"image/png"
	"os"
)

// PixmapOptions controls rendering.
type PixmapOptions struct {
	Zoom  float64 // scale factor (1.0 = 72 DPI); <=0 means 1.0
	Gray  bool    // render in grayscale instead of RGB
	Alpha bool    // include an alpha channel
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

// Pixmap renders the page.
func (p *Page) Pixmap(opts ...PixmapOptions) (*Pixmap, error) {
	o := PixmapOptions{Zoom: 1}
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Zoom <= 0 {
		o.Zoom = 1
	}
	blob, err := p.pixmapRaw(o.Zoom, o.Gray, o.Alpha)
	if err != nil {
		return nil, err
	}
	if len(blob) < 20 {
		return nil, errors.New("gomupdf: short pixmap blob")
	}
	rd := func(i int) int { return int(int32(binary.LittleEndian.Uint32(blob[i*4 : i*4+4]))) }
	pm := &Pixmap{
		Width:   rd(0),
		Height:  rd(1),
		N:       rd(2),
		Stride:  rd(3),
		Alpha:   rd(4) != 0,
		Samples: blob[20:],
	}
	return pm, nil
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
