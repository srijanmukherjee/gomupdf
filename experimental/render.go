package experimental

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/srijanmukherjee/gomupdf"
)

// RenderOption configures rasterization.
type RenderOption func(*renderConfig)

type renderConfig struct {
	zoom float64
	gray bool
}

// DPI sets the render resolution. 72 DPI == zoom 1.0.
func DPI(dpi float64) RenderOption {
	return func(c *renderConfig) {
		if dpi > 0 {
			c.zoom = dpi / 72
		}
	}
}

// Zoom sets the render scale factor directly (1.0 == 72 DPI).
func Zoom(z float64) RenderOption {
	return func(c *renderConfig) { c.zoom = z }
}

// Grayscale renders in grayscale instead of RGB.
func Grayscale() RenderOption {
	return func(c *renderConfig) { c.gray = true }
}

func (p *Page) pixmap(opts []RenderOption) (*gomupdf.Pixmap, error) {
	cfg := renderConfig{zoom: 1}
	for _, o := range opts {
		o(&cfg)
	}
	return p.raw.Pixmap(gomupdf.PixmapOptions{Zoom: cfg.zoom, Gray: cfg.gray})
}

// Image renders the page to a standard library image.Image.
func (p *Page) Image(opts ...RenderOption) (image.Image, error) {
	pm, err := p.pixmap(opts)
	if err != nil {
		return nil, err
	}
	return pm.Image()
}

// PNG renders the page and returns encoded PNG bytes.
func (p *Page) PNG(opts ...RenderOption) ([]byte, error) {
	pm, err := p.pixmap(opts)
	if err != nil {
		return nil, err
	}
	return pm.PNG()
}

// SavePNG renders the page and writes it to path as PNG.
func (p *Page) SavePNG(path string, opts ...RenderOption) error {
	pm, err := p.pixmap(opts)
	if err != nil {
		return err
	}
	return pm.SavePNG(path)
}

// SavePNGs renders every page into dir as page-1.png, page-2.png, ... and
// returns the written paths. dir is created if needed.
func (d *Doc) SavePNGs(dir string, opts ...RenderOption) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var paths []string
	for i, page := range d.Pages() {
		path := filepath.Join(dir, fmt.Sprintf("page-%d.png", i+1))
		if err := page.SavePNG(path, opts...); err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// Thumbnail renders the first page to an image.Image (default options unless
// overridden, e.g. experimental.Zoom(0.3)).
func (d *Doc) Thumbnail(opts ...RenderOption) (image.Image, error) {
	p, err := d.Page(0)
	if err != nil {
		return nil, err
	}
	return p.Image(opts...)
}
