package gopdf

import (
	"io"
	"math"
	"strings"

	"github.com/signintech/gopdf"
)

type Impl struct {
	Width  float64
	Height float64
	GoPdf  *gopdf.GoPdf

	// TODO: remove these unused color fields once it's confirmed nothing relies on them.
	textR, textG, textB       uint8 //nolint:unused
	fillR, fillG, fillB       uint8
	strokeR, strokeG, strokeB uint8 //nolint:unused
}

func (f Impl) maybeAddPage(lineHeight float64) {
	Y := f.GoPdf.GetY()
	H := f.Height
	TopM := f.GoPdf.MarginTop()
	BottomM := f.GoPdf.MarginBottom()

	if math.Mod(Y, H) > H-TopM-BottomM-lineHeight {
		f.GoPdf.AddPage()
	}
}

// Add a new page
func (f Impl) AddPage() {
	f.GoPdf.AddPage()
}

// Position
func (f Impl) GetX() float64 {
	return f.GoPdf.GetX()
}

func (f Impl) GetY() float64 {
	return f.GoPdf.GetY()
}

func (f Impl) SetX(x float64) {
	f.GoPdf.SetX(x)
}

func (f Impl) SetY(y float64) {
	f.GoPdf.SetY(y)
}

// Page size
func (f Impl) GetPageSize() (width float64, height float64) {
	return f.Width, f.Height
}

// Margins
func (f Impl) SetMarginLeft(margin float64) {
	f.GoPdf.SetMarginLeft(margin)
}

func (f Impl) SetMarginRight(margin float64) {
	f.GoPdf.SetMarginRight(margin)
}

func (f Impl) SetMarginTop(margin float64) {
	f.GoPdf.SetTopMargin(margin)
}

func (f Impl) SetFont(family string, style string, size int) error {
	return f.GoPdf.SetFont(family, style, size)
}

// Writing
func (f Impl) WriteText(h float64, text string) {
	f.maybeAddPage(h)
	w, _ := f.GoPdf.MeasureTextWidth(text)
	err := f.GoPdf.CellWithOption(&gopdf.Rect{W: w, H: h}, text, gopdf.CellOption{
		Align: gopdf.Left,
		Float: gopdf.Right,
	})
	if err != nil {
		panic(err)
	}
}

func (f Impl) CellFormat(w float64, h float64, txtStr string, borderStr string, ln int, alignStr string, fill bool, link int, linkStr string) {
	f.maybeAddPage(h)
	f.GoPdf.SetFillColor(f.fillR, f.fillG, f.fillB)
	borderStr = strings.ToUpper(borderStr)

	var border int
	if borderStr == "1" {
		border = gopdf.AllBorders
	} else {
		if strings.Contains(borderStr, "L") {
			border = border | gopdf.Left
		}
		if strings.Contains(borderStr, "R") {
			border = border | gopdf.Right
		}
		if strings.Contains(borderStr, "T") {
			border = border | gopdf.Top
		}
		if strings.Contains(borderStr, "B") {
			border = border | gopdf.Bottom
		}
	}

	float := gopdf.Right
	if ln == 1 {
		float = gopdf.Bottom
	}

	x := f.GoPdf.GetX()
	y := f.GoPdf.GetY()
	f.GoPdf.RectFromUpperLeftWithStyle(x, y, w, h, "F")

	err := f.GoPdf.CellWithOption(&gopdf.Rect{W: w, H: h}, txtStr, gopdf.CellOption{
		Align:  gopdf.Left,
		Border: border,
		Float:  float,
	})
	if err != nil {
		panic(err)
	}
}

func (f Impl) AddInternalLink(anchor string) {
	f.GoPdf.SetAnchor(anchor)
}

func (f Impl) WriteInternalLink(h float64, text string, anchor string) {
	f.maybeAddPage(h)
	x := f.GoPdf.GetX()
	y := f.GoPdf.GetY()
	w, _ := f.GoPdf.MeasureTextWidth(text)
	err := f.GoPdf.CellWithOption(&gopdf.Rect{W: w, H: h}, text, gopdf.CellOption{
		Align: gopdf.Left,
		Float: gopdf.Right,
	})
	if err != nil {
		panic(err)
	}
	f.GoPdf.AddExternalLink(anchor, x, y, w, h)
}

func (f Impl) WriteExternalLink(h float64, text string, destination string) {
	f.maybeAddPage(h)
	x := f.GoPdf.GetX()
	y := f.GoPdf.GetY()
	w, _ := f.GoPdf.MeasureTextWidth(text)
	err := f.GoPdf.CellWithOption(&gopdf.Rect{W: w, H: h}, text, gopdf.CellOption{
		Align: gopdf.Left,
		Float: gopdf.Right,
	})
	if err != nil {
		panic(err)
	}
	f.GoPdf.AddExternalLink(destination, x, y, w, h)
}

func (f Impl) BR(height float64) {
	f.maybeAddPage(height)
	f.GoPdf.Br(height)
}

// Images
func (f Impl) RegisterImage(id string, format string, src io.Reader) {
	// f.GoPdf.RegisterImageOptionsReader(id, gofpdf.ImageOptions{ImageType: format}, src)
}

func (f Impl) UseImage(imgID string, x, y, w, h float64) {
	// f.checkNewPage()
	// f.GoPdf.ImageOptions(imgID, x, y, w, h, false, gofpdf.ImageOptions{ImageType: "", ReadDpi: true}, 0, "")
}

// Measuring
func (f Impl) MeasureTextWidth(text string) float64 {
	width, err := f.GoPdf.MeasureTextWidth(text)
	if err != nil {
		panic(err)
	}
	return width
}

func (f Impl) SplitText(text string, width float64) []string {
	gp := f.GoPdf
	var lineText []rune
	var lineTexts []string
	utf8Texts := []rune(text)
	utf8TextsLen := len(utf8Texts) // utf8 string quantity
	if utf8TextsLen == 0 {
		return lineTexts
	}
	for i := 0; i < utf8TextsLen; i++ {
		lineWidth, err := gp.MeasureTextWidth(string(lineText))
		if err != nil {
			panic(err)
		}
		runeWidth, err := gp.MeasureTextWidth(string(utf8Texts[i]))
		if err != nil {
			panic(err)
		}
		if lineWidth+runeWidth > width && utf8Texts[i] != '\n' {
			lineTexts = append(lineTexts, string(lineText))
			lineText = lineText[0:0]
			continue
		}
		if utf8Texts[i] == '\n' {
			lineTexts = append(lineTexts, string(lineText))
			lineText = lineText[0:0]
			continue
		}
		if i == utf8TextsLen-1 {
			lineText = append(lineText, utf8Texts[i])
			lineTexts = append(lineTexts, string(lineText))
		}
		lineText = append(lineText, utf8Texts[i])

	}
	return lineTexts
}

// Colors
// TODO: value receivers make these field assignments ineffective; switch to pointer
// receivers (or drop the fields) once the dependent code paths are verified.
func (f Impl) SetDrawColor(r uint8, g uint8, b uint8) {
	f.strokeR, f.strokeB, f.strokeG = r, g, b //nolint:staticcheck // SA4005: see TODO above
	f.GoPdf.SetStrokeColor(r, g, b)
}

func (f Impl) SetFillColor(r uint8, g uint8, b uint8) {
	f.fillR, f.fillB, f.fillG = r, g, b //nolint:staticcheck // SA4005: see TODO above
	f.GoPdf.SetFillColor(r, g, b)
}

func (f Impl) SetTextColor(r uint8, g uint8, b uint8) {
	f.textR, f.textB, f.textG = r, g, b //nolint:staticcheck // SA4005: see TODO above
	f.GoPdf.SetTextColor(r, g, b)
}

// Width
func (f Impl) SetLineWidth(width float64) {
	f.GoPdf.SetLineWidth(width)
}

func (f Impl) Line(x1 float64, y1 float64, x2 float64, y2 float64) {
	f.GoPdf.Line(x1, y1, x2, y2)
}

func (f Impl) Circle(x, y, r float64) {
	// Oval is stroke-only, so we fake a fill with a stroked band: path-radius r/2 stroked at line width
	// r yields a band from the center out to full radius r (inner hole = r/2 - r/2 = 0).
	f.GoPdf.SetLineWidth(r)
	f.GoPdf.Oval(x-r/2, y-r/2, x+r/2, y+r/2)
}

func (f Impl) Write(w io.Writer) error {
	_, err := f.GoPdf.WriteTo(w)
	return err
}

func (f Impl) GetMargins() (left, top, right, bottom float64) {
	return f.GoPdf.Margins()
}

func (f Impl) AddFont(family string, styleStr string, data []byte) error {
	styleStr = strings.ToUpper(styleStr)

	style := gopdf.Regular
	if strings.Contains(styleStr, "B") {
		style = style | gopdf.Bold
	}
	if strings.Contains(styleStr, "I") {
		style = style | gopdf.Italic
	}
	if strings.Contains(styleStr, "U") {
		style = style | gopdf.Underline
	}

	return f.GoPdf.AddTTFFontDataWithOption(family, data, gopdf.TtfOption{
		Style: style,
	})
}
