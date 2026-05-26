package gopdf

import (
	"context"
	"io"

	"github.com/go-swiss/fonts"
	pdf "github.com/stephenafamo/goldmark-pdf"
)

// gofpdf-equivalent default margins, in points: 1cm on L/T/R and 2cm on the
// bottom (gofpdf's auto-page-break trigger sits at 2*margin). Matching these
// keeps the gopdf and fpdf backends visually aligned.
const (
	defaultMarginLTR    = 28.35
	defaultMarginBottom = 56.70
)

type Config struct {
	Title   string
	Subject string

	// Orientation accepts "P"/"Portrait" or "L"/"Landscape" (case-insensitive).
	// Empty and unknown values default to portrait.
	Orientation string

	// PaperSize selects a named page size: "A3", "A4", "A5", "Letter",
	// "Legal", "Tabloid" (case-insensitive). Empty defaults to "A4". This
	// mirrors fpdf.Config.PaperSize
	PaperSize string

	// Logo, when set, is auto-registered under the image ID "logo" before the
	// first page is added. HeaderFunc/FooterFunc closures can then draw it
	// with impl.UseImage("logo", ...) without having to call RegisterImage
	// themselves. LogoFormat is passed straight through to RegisterImage.
	Logo       io.Reader
	LogoFormat string

	// HeaderFunc and FooterFunc are invoked by gopdf on every
	// AddPage (the underlying lib fires both at page-open). The returned
	// closure is what actually draws; the outer function exists so callers
	// can capture the wrapper impl and fonts cache they need at draw time.
	//
	// Mirrors the fpdf backend's HeaderFunc/FooterFunc shape, but receives
	// *Impl (not a value copy) because gopdf's wrapper tracks mirrored
	// graphics state through pointer receivers — passing a value would
	// silently drop those mutations.
	HeaderFunc func(impl *Impl, fontsCache fonts.Cache) func()
	FooterFunc func(impl *Impl, fontsCache fonts.Cache) func()
}

// Deprecated: Use New. Kept for backwards compatibility; supplies a nil
// fonts cache, which is fine as long as HeaderFunc/FooterFunc don't try to
// load fonts through it.
func WithGoPdf(ctx context.Context, c Config) pdf.Option {
	return pdf.WithPDF(New(ctx, c, nil))
}
