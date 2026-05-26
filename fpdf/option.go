package fpdf

import (
	"io"

	"github.com/go-swiss/fonts"
)

type Config struct {
	Title   string
	Subject string

	Orientation string // Default Portrait
	PaperSize   string // Default A4

	Logo       io.Reader
	LogoFormat string

	// HeaderFunc and FooterFunc receive *Impl so that header/footer closures
	// running on backends with mirrored graphics state (notably the gopdf
	// backend) can mutate it through the wrapper. fpdf has nothing to track
	// today, but the signature is unified across backends to keep the same
	// closure literal portable between WithFpdf and WithGoPdf.
	HeaderFunc func(impl *Impl, fontsCache fonts.Cache) func()
	FooterFunc func(impl *Impl, fontsCache fonts.Cache) func()
}
