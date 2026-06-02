package experimental

import (
	"fmt"
	"os"
)

// Image is an embedded image extracted from a page, tagged with its page and
// fill-order index, carrying its original encoded bytes and placement region.
type Image struct {
	Page   int
	Index  int
	Ext    string // jpeg, png, jpx, ...
	Bytes  []byte // original encoded bytes
	Width  int    // source pixel width
	Height int    // source pixel height
	Region Rect   // placement rectangle on the page
}

// Save writes the image's encoded bytes to path.
func (im Image) Save(path string) error { return os.WriteFile(path, im.Bytes, 0o644) }

// Images extracts every embedded image on the page, preserving original
// encoding where available.
func (p *Page) Images() ([]Image, error) {
	infos, err := p.raw.GetImages()
	if err != nil {
		return nil, err
	}
	out := make([]Image, 0, len(infos))
	for _, info := range infos {
		ex, err := p.raw.ExtractImage(info.Index)
		if err != nil {
			return nil, err
		}
		if ex == nil {
			continue
		}
		out = append(out, Image{
			Page:   p.idx,
			Index:  info.Index,
			Ext:    ex.Ext,
			Bytes:  ex.Bytes,
			Width:  info.Width,
			Height: info.Height,
			Region: rectFromGeometry(info.BBox),
		})
	}
	return out, nil
}

// Images extracts embedded images across every page, tagged by page.
func (d *Doc) Images() ([]Image, error) {
	var out []Image
	for _, page := range d.Pages() {
		imgs, err := page.Images()
		if err != nil {
			return nil, err
		}
		out = append(out, imgs...)
	}
	return out, nil
}

// SaveImages extracts every image and writes them into dir as
// page<P>-img<I>.<ext>, returning the written paths. dir is created if needed.
func (d *Doc) SaveImages(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	imgs, err := d.Images()
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, im := range imgs {
		path := fmt.Sprintf("%s/page%d-img%d.%s", dir, im.Page+1, im.Index, im.Ext)
		if err := im.Save(path); err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
