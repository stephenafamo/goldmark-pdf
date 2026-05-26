package fpdf

import (
	"context"
	"io"
	"strings"

	"github.com/go-swiss/fonts"
	"github.com/phpdave11/gofpdf"
)

// gofpdf panics on runes above the BMP (its Cw array is 65536 entries), so
// substitute non-BMP runes with U+FFFD. Proper fix is the gopdf
// backend (gopdf/), which handles them natively but is incomplete.
// See https://github.com/stephenafamo/goldmark-pdf/issues/27.
const (
	maxSupportedRune = 0xFFFF
	unknownRune      = '�' // �
)

func sanitizeUnicode(s string) string {
	for _, r := range s {
		if r > maxSupportedRune {
			var b strings.Builder
			b.Grow(len(s))
			for _, r := range s {
				if r > maxSupportedRune {
					b.WriteRune(unknownRune)
				} else {
					b.WriteRune(r)
				}
			}
			return b.String()
		}
	}
	return s
}

func New(ctx context.Context, c Config, fontsCache fonts.Cache) *Impl {
	fpdf := Impl{
		Fpdf:    gofpdf.New(c.Orientation, "pt", c.PaperSize, ""),
		anchors: make(map[string]int),
	}

	fpdf.Fpdf.SetTitle(c.Title, true)
	fpdf.Fpdf.SetSubject(c.Subject, true)
	fpdf.Fpdf.SetCellMargin(0)

	// Pre-register the logo so HeaderFunc/FooterFunc closures can pull it
	// up with impl.UseImage("logo", ...) without having to call
	// RegisterImage themselves. Mirrors the gopdf backend's behavior.
	if c.Logo != nil {
		fpdf.RegisterImage("logo", c.LogoFormat, c.Logo)
	}

	if c.HeaderFunc != nil {
		fpdf.Fpdf.SetHeaderFunc(c.HeaderFunc(&fpdf, fontsCache))
	}

	if c.FooterFunc != nil {
		fpdf.Fpdf.SetFooterFunc(c.FooterFunc(&fpdf, fontsCache))
	}

	fpdf.AddPage()

	return &fpdf
}

type internalLink struct {
	page          int
	x, y          float64
	width, height float64
	anchor        string
}

type Impl struct {
	Fpdf        *gofpdf.Fpdf
	anchorLinks []internalLink
	anchors     map[string]int
}

// Add a new page
func (f Impl) AddPage() {
	f.Fpdf.AddPage()
}

// Position
func (f Impl) GetX() float64 {
	return f.Fpdf.GetX()
}

func (f Impl) GetY() float64 {
	return f.Fpdf.GetY()
}

func (f Impl) SetX(x float64) {
	f.Fpdf.SetX(x)
}

func (f Impl) SetY(y float64) {
	f.Fpdf.SetY(y)
}

// Page size
func (f Impl) GetPageSize() (width float64, height float64) {
	return f.Fpdf.GetPageSize()
}

// Margins
func (f Impl) SetMarginLeft(margin float64) {
	f.Fpdf.SetLeftMargin(margin)
}

func (f Impl) SetMarginRight(margin float64) {
	f.Fpdf.SetRightMargin(margin)
}

func (f Impl) SetMarginTop(margin float64) {
	f.Fpdf.SetTopMargin(margin)
}

// SetMarginBottom sets the bottom page margin via gofpdf's auto-page-break
// trigger — that's the only way gofpdf exposes the bottom margin, since
// gofpdf.SetMargins only accepts left/top/right. Always re-enables auto
// page break so the renderer's page-break expectations hold.
func (f Impl) SetMarginBottom(margin float64) {
	f.Fpdf.SetAutoPageBreak(true, margin)
}

func (f Impl) SetFont(family string, style string, size int) error {
	f.Fpdf.SetFont(family, style, float64(size))
	return nil
}

// Writing
func (f Impl) WriteText(height float64, text string) {
	f.Fpdf.Write(height, sanitizeUnicode(text))
}

func (f Impl) CellFormat(w float64, h float64, txtStr string, borderStr string, ln int, alignStr string, fill bool, link int, linkStr string) {
	f.Fpdf.SetCellMargin(0)
	f.Fpdf.CellFormat(w, h, sanitizeUnicode(txtStr), borderStr, ln, alignStr, fill, link, linkStr)
}

func (f *Impl) AddInternalLink(anchor string) {
	linkID := f.Fpdf.AddLink()
	f.Fpdf.SetLink(linkID, f.GetY(), -1)
	f.anchors[anchor] = linkID
}

func (f *Impl) WriteInternalLink(lineHeight float64, text string, anchor string) {
	text = sanitizeUnicode(text)
	f.anchorLinks = append(f.anchorLinks, internalLink{
		page:   f.Fpdf.PageNo(),
		width:  f.MeasureTextWidth(text),
		height: lineHeight,
		x:      f.GetX(),
		y:      f.GetY(),
		anchor: anchor,
	})
	f.Fpdf.WriteLinkString(lineHeight, text, "#"+anchor)
}

func (f Impl) WriteExternalLink(lineHeight float64, text string, destination string) {
	f.Fpdf.WriteLinkString(lineHeight, sanitizeUnicode(text), destination)
}

func (f Impl) BR(height float64) {
	f.Fpdf.Ln(height)
}

// Images
func (f Impl) RegisterImage(id string, format string, src io.Reader) {
	f.Fpdf.RegisterImageOptionsReader(id, gofpdf.ImageOptions{ImageType: format, ReadDpi: false}, src)
}

func (f Impl) UseImage(imgID string, x, y, w, h float64) {
	f.Fpdf.ImageOptions(imgID, x, y, w, h, true, gofpdf.ImageOptions{ImageType: "", ReadDpi: false}, 0, "")
}

// Measuring
func (f Impl) MeasureTextWidth(text string) float64 {
	return f.Fpdf.GetStringWidth(sanitizeUnicode(text))
}

func (f Impl) SplitText(txt string, w float64) []string {
	lines := f.Fpdf.SplitLines([]byte(sanitizeUnicode(txt)), w)

	split := make([]string, len(lines))
	for k, line := range lines {
		split[k] = string(line)
	}

	return split
}

// Colors
func (f Impl) SetDrawColor(r uint8, g uint8, b uint8) {
	f.Fpdf.SetDrawColor(int(r), int(g), int(b))
}

func (f Impl) SetFillColor(r uint8, g uint8, b uint8) {
	f.Fpdf.SetFillColor(int(r), int(g), int(b))
}

func (f Impl) SetTextColor(r uint8, g uint8, b uint8) {
	f.Fpdf.SetTextColor(int(r), int(g), int(b))
}

// Width
func (f Impl) SetLineWidth(width float64) {
	f.Fpdf.SetLineWidth(width)
}

func (f Impl) Line(x1 float64, y1 float64, x2 float64, y2 float64) {
	f.Fpdf.Line(x1, y1, x2, y2)
}

func (f Impl) Circle(x, y, r float64) {
	f.Fpdf.Circle(x, y, r, "F")
}

func (f Impl) Write(w io.Writer) error {
	// write the internal links
	for _, link := range f.anchorLinks {
		id, ok := f.anchors[link.anchor]
		if !ok {
			continue
		}

		f.Fpdf.SetPage(link.page)
		f.Fpdf.Link(link.x, link.y, link.width, link.height, id)
	}

	f.Fpdf.SetPage(f.Fpdf.PageCount())
	return f.Fpdf.Output(w)
}

func (f Impl) GetMargins() (left, top, right, bottom float64) {
	return f.Fpdf.GetMargins()
}

func (f Impl) AddFont(family string, style string, data []byte) error {
	f.Fpdf.AddUTF8FontFromBytes(family, style, data)
	return nil
}
