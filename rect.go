package gomupdf

// Rect is a bounding box in PDF points (origin top-left, y grows downward),
// expressed as a position and size. It is the box type carried by extracted
// words, spans, and blocks.
type Rect struct {
	X, Y, W, H float64
}

// X1 returns the right edge (X + W).
func (r Rect) X1() float64 { return r.X + r.W }

// Y1 returns the bottom edge (Y + H).
func (r Rect) Y1() float64 { return r.Y + r.H }

// CenterY returns the vertical midpoint, useful when clustering boxes into rows.
func (r Rect) CenterY() float64 { return r.Y + r.H/2 }
