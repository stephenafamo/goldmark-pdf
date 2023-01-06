package pdf

import (
	"io"
)

type PDF interface {
	// Add a new page
	AddPage()

	// Position
	GetX() float64
	GetY() float64

	SetX(x float64)
	SetY(y float64)

	// Page size
	GetPageSize() (width, height float64)

	SetMarginLeft(margin float64)
	SetMarginRight(margin float64)
	GetMargins() (left, top, right, bottom float64)

	// Font
	AddFont(family string, style string, data []byte) error
	SetFont(family string, style string, size int) error

	// Writing
	WriteText(height float64, text string)
	CellFormat(w, h float64, txtStr, borderStr string, ln int, alignStr string, fill bool, link int, linkStr string)
	BR(h float64)

	// Links
	// Add an internal link anchor to the current position
	AddInternalLink(anchor string)
	// record an internal ink to the given anchor
	WriteInternalLink(lineHeight float64, text string, anchor string)
	WriteExternalLink(lineHeight float64, text, destination string)

	// Images
	RegisterImage(id, format string, src io.Reader)
	UseImage(id string, x, y, w, h float64)

	// Measuring
	MeasureTextWidth(text string) float64
	SplitText(txt string, w float64) []string

	// Colors
	SetDrawColor(r uint8, g uint8, b uint8)
	SetFillColor(r uint8, g uint8, b uint8)
	SetTextColor(r uint8, g uint8, b uint8)

	// Width
	SetLineWidth(width float64)
	Line(x1, x2, y1, y2 float64)
	// Rect(x1, x2, y1, y2 float64)

	// Rendering
	Write(io.Writer) error
}
