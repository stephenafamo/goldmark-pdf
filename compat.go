package pdf

import (
	"context"

	"github.com/go-swiss/fonts"

	"github.com/stephenafamo/goldmark-pdf/fpdf"
)

type (
	// Deprecated: Use fpdf.Impl
	Fpdf = fpdf.Impl
	// Deprecated: Use fpdf.Config
	FpdfConfig = fpdf.Config
)

// Deprecated: Use fpdf.New
func NewFpdf(ctx context.Context, c FpdfConfig, fontsCache fonts.Cache) *Fpdf {
	return fpdf.New(ctx, c, fontsCache)
}
