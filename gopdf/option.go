package gopdf

import (
	"context"

	"github.com/signintech/gopdf"
	pdf "github.com/stephenafamo/goldmark-pdf"
)

type InitConfig struct {
	Title   string
	Subject string

	PaperSize *gopdf.Rect // Default A4
}

func WithGoPdf(ctx context.Context, c InitConfig) pdf.Option {
	if c.PaperSize == nil {
		c.PaperSize = gopdf.PageSizeA4
	}

	gpdf := Impl{
		GoPdf:  &gopdf.GoPdf{},
		Width:  c.PaperSize.W,
		Height: c.PaperSize.H,
	}

	gpdf.GoPdf.Start(gopdf.Config{
		Unit:     gopdf.UnitPT,
		PageSize: *c.PaperSize,
	})

	gpdf.GoPdf.SetInfo(gopdf.PdfInfo{
		Title:   c.Title,
		Subject: c.Subject,
	})

	// if c.HeaderStyle == (pdf.Style{}) {
	// c.HeaderStyle = pdf.Style{Font: pdf.FontRoboto, Size: 12, Spacing: 10,
	// TextColor: color.RGBA{10, 10, 10, 0}, FillColor: color.RGBA{255, 255, 255, 0}}
	// }

	// if c.FooterStyle == (pdf.Style{}) {
	// c.FooterStyle = pdf.Style{Font: pdf.FontRoboto, Size: 11, Spacing: 2,
	// TextColor: color.RGBA{10, 10, 10, 0}, FillColor: color.RGBA{255, 255, 255, 0}}
	// }

	// pdf.AddFonts(ctx, gpdf, []pdf.Font{c.HeaderStyle.Font, c.FooterStyle.Font})

	// if c.Logo != nil {
	// gpdf.RegisterImage("logo", c.LogoFormat, c.Logo)
	// }

	// mleft, mtop, mright, mbottom := gpdf.GetMargins()
	// pageWidth, _ := gpdf.GetPageSize()

	// LH := c.HeaderStyle.Size + c.HeaderStyle.Spacing

	// if c.HeaderText != "" {
	// gpdf.GoPdf.SetHeaderFunc(func() {
	// pdf.SetStyle(gpdf, c.HeaderStyle)

	// gpdf.SetX(mleft + 5)

	// if c.Logo != nil {
	// var logoWidth float64 = c.HeaderStyle.Size
	// gpdf.SetX(gpdf.GetX() + logoWidth)
	// gpdf.UseImage("logo", mleft, mtop, logoWidth, logoWidth)
	// }

	// gpdf.CellFormat(0, 0, c.HeaderText, "", 0, "LT", false, 0, "")
	// gpdf.WriteText(LH, "\n")
	// })
	// }

	// if c.FooterText != "" {
	// gpdf.GoPdf.SetFooterFunc(func() {
	// pdf.SetStyle(gpdf, c.FooterStyle)
	// gpdf.SetY(-mbottom)
	// gpdf.GoPdf.Ln(-1)
	// gpdf.GoPdf.Ln(-1)
	// gpdf.SetX(mleft)
	// gpdf.CellFormat(0, 0, fmt.Sprintf("Page %d", gpdf.GoPdf.PageNo()), "", 0, "LB", false, 0, "")
	// gpdf.SetX(mleft)
	// gpdf.CellFormat(pageWidth-mleft-mright, 0, c.FooterText, "", 0, "RB", false, 0, "")
	// })
	// }

	gpdf.AddPage()

	return pdf.WithPDF(gpdf)
}
