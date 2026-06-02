package gomupdf

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Word is a whitespace-delimited word with its character-derived bounding box.
// BBox is the union of the word's character boxes, giving precise geometry for
// table reconstruction and column/row clustering.
type Word struct {
	Text  string
	BBox  Rect
	Block int
	Line  int
}

// Words returns the page's words with character-level bounding boxes. These
// boxes are more precise than the span-split approximations and are the
// recommended input for FindTables.
func (p *Page) Words() ([]Word, error) {
	raw, err := p.wordsRaw()
	if err != nil {
		return nil, err
	}
	var out []Word
	for _, ln := range strings.Split(raw, "\n") {
		if ln == "" {
			continue
		}
		tab := strings.IndexByte(ln, '\t')
		if tab < 0 {
			continue
		}
		f := strings.Fields(ln[:tab])
		if len(f) != 6 {
			continue
		}
		x0, _ := strconv.ParseFloat(f[0], 64)
		y0, _ := strconv.ParseFloat(f[1], 64)
		x1, _ := strconv.ParseFloat(f[2], 64)
		y1, _ := strconv.ParseFloat(f[3], 64)
		blk, _ := strconv.Atoi(f[4])
		lno, _ := strconv.Atoi(f[5])
		out = append(out, Word{
			Text:  ln[tab+1:],
			BBox:  Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0},
			Block: blk,
			Line:  lno,
		})
	}
	return out, nil
}

// Span is a positioned run of text: its text, bounding box, baseline origin,
// and font.
type Span struct {
	Text     string
	BBox     Rect
	OriginX  float64
	OriginY  float64
	FontName string
	FontSize float64
}

// Block is a text or image block containing zero or more spans.
type Block struct {
	Type  string // "text" | "image"
	BBox  Rect
	Spans []Span
}

// rawBBox mirrors the structured-text {x,y,w,h} box.
type rawBBox struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type rawFont struct {
	Name string  `json:"name"`
	Size float64 `json:"size"`
}

type rawLine struct {
	BBox rawBBox `json:"bbox"`
	Font rawFont `json:"font"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Text string  `json:"text"`
}

type rawBlock struct {
	Type  string    `json:"type"`
	BBox  rawBBox   `json:"bbox"`
	Lines []rawLine `json:"lines"`
}

type rawStext struct {
	Blocks []rawBlock `json:"blocks"`
}

// StructuredText returns the page's blocks and spans with bounding boxes. Spans
// are positioned text fragments; clustering them by x yields columns and by y
// yields rows, which is how callers needing exact column geometry reconstruct
// tables rather than relying on reading-order text.
func (p *Page) StructuredText() ([]Block, error) {
	js, err := p.StructuredJSON()
	if err != nil {
		return nil, err
	}
	var raw rawStext
	if err := json.Unmarshal([]byte(js), &raw); err != nil {
		return nil, err
	}
	blocks := make([]Block, 0, len(raw.Blocks))
	for _, rb := range raw.Blocks {
		b := Block{Type: rb.Type, BBox: Rect(rb.BBox)}
		for _, rl := range rb.Lines {
			b.Spans = append(b.Spans, Span{
				Text:     rl.Text,
				BBox:     Rect(rl.BBox),
				OriginX:  rl.X,
				OriginY:  rl.Y,
				FontName: rl.Font.Name,
				FontSize: rl.Font.Size,
			})
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// Spans returns all text spans on the page in document order — the flat
// positioned-text stream that table reconstruction typically starts from.
func (p *Page) Spans() ([]Span, error) {
	blocks, err := p.StructuredText()
	if err != nil {
		return nil, err
	}
	var out []Span
	for _, b := range blocks {
		if b.Type == "text" {
			out = append(out, b.Spans...)
		}
	}
	return out, nil
}
