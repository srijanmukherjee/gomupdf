package gomupdf

import (
	"strconv"
	"strings"

	"github.com/srijanmukherjee/gomupdf/geometry"
)

// ImageInfo describes an image placed on a page. BBox is the placement
// rectangle in page points.
type ImageInfo struct {
	Index  int
	BBox   geometry.Rect
	Width  int // source image pixel width
	Height int // source image pixel height
	BPC    int // bits per component
	N      int // colorspace components (0 if none)
	Ext    string
}

// GetImages returns the images drawn on the page, in fill order, each with its
// placement bbox and source dimensions.
func (p *Page) GetImages() ([]ImageInfo, error) {
	raw, err := p.imagesRaw()
	if err != nil {
		return nil, err
	}
	var out []ImageInfo
	idx := 0
	for _, ln := range strings.Split(raw, "\n") {
		f := strings.Fields(ln)
		if len(f) != 10 || f[0] != "IMG" {
			continue
		}
		x0, _ := strconv.ParseFloat(f[1], 64)
		y0, _ := strconv.ParseFloat(f[2], 64)
		x1, _ := strconv.ParseFloat(f[3], 64)
		y1, _ := strconv.ParseFloat(f[4], 64)
		w, _ := strconv.Atoi(f[5])
		h, _ := strconv.Atoi(f[6])
		bpc, _ := strconv.Atoi(f[7])
		n, _ := strconv.Atoi(f[8])
		out = append(out, ImageInfo{
			Index:  idx,
			BBox:   geometry.Rect{X0: x0, Y0: y0, X1: x1, Y1: y1},
			Width:  w,
			Height: h,
			BPC:    bpc,
			N:      n,
			Ext:    f[9],
		})
		idx++
	}
	return out, nil
}

// ExtractedImage is the encoded bytes of an image plus its file extension.
type ExtractedImage struct {
	Ext   string // jpeg | png | jpx | …
	Bytes []byte
}

// ExtractImage returns the encoded bytes of the index-th image on the page.
// Original encoding is preserved when available (e.g. JPEG); uncompressed
// images are re-encoded as PNG. Returns nil if there is no image at that index.
func (p *Page) ExtractImage(index int) (*ExtractedImage, error) {
	data, ext, err := p.imageBytesRaw(index)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	return &ExtractedImage{Ext: ext, Bytes: data}, nil
}
