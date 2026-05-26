// Package main renders every markdown file in this directory through both
// the fpdf and gopdf backends. Outputs are written next to the input as
// <name>.<backend>.pdf (e.g. tables.md → tables.fpdf.pdf, tables.gopdf.pdf).
// PDFs are .gitignored.
//
// Run with: go run ./examples
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-swiss/fonts"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	pdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/stephenafamo/goldmark-pdf/fpdf"
	"github.com/stephenafamo/goldmark-pdf/gopdf"
)

// exampleConfig holds optional per-example hooks. Both backends now share
// the same *Impl-pointer HeaderFunc/FooterFunc signature, so closures can
// be written once per backend and dropped into either Config. The zero
// value (returned by the map lookup for unknown keys) leaves every hook
// nil.
type exampleConfig struct {
	fpdfHeader  func(impl *fpdf.Impl, fc fonts.Cache) func()
	fpdfFooter  func(impl *fpdf.Impl, fc fonts.Cache) func()
	gopdfHeader func(impl *gopdf.Impl, fc fonts.Cache) func()
	gopdfFooter func(impl *gopdf.Impl, fc fonts.Cache) func()
}

// exampleConfigs keys on the .md basename. Add an entry to enable hooks
// for a new example.
var exampleConfigs = map[string]exampleConfig{
	"header-footer": {
		fpdfHeader:  fpdfTitleBar,
		fpdfFooter:  fpdfPageNumber,
		gopdfHeader: gopdfTitleBar,
		gopdfFooter: gopdfPageNumber,
	},
}

// fpdfTitleBar draws "Header / Footer Sample" in the top page margin.
// Helvetica is gofpdf's built-in font so this works on page 1 without any
// font-loading prep.
func fpdfTitleBar(impl *fpdf.Impl, _ fonts.Cache) func() {
	return func() {
		_ = impl.SetFont("Helvetica", "B", 9)
		ml, mt, mr, _ := impl.GetMargins()
		pw, _ := impl.GetPageSize()
		impl.Fpdf.SetXY(ml, mt/2)
		impl.SetTextColor(80, 80, 80)
		impl.CellFormat(pw-ml-mr, 14, "Header / Footer Sample", "B", 0, "L", false, 0, "")
	}
}

// fpdfPageNumber draws "Page N" centered in the bottom page margin.
func fpdfPageNumber(impl *fpdf.Impl, _ fonts.Cache) func() {
	return func() {
		_ = impl.SetFont("Helvetica", "", 8)
		ml, _, mr, mb := impl.GetMargins()
		pw, ph := impl.GetPageSize()
		impl.Fpdf.SetXY(ml, ph-(mb/2)-10)
		impl.SetTextColor(80, 80, 80)
		impl.CellFormat(pw-ml-mr, 10, fmt.Sprintf("Page %d", impl.Fpdf.PageNo()),
			"T", 0, "C", false, 0, "")
	}
}

func gopdfTitleBar(impl *gopdf.Impl, _ fonts.Cache) func() {
	return func() {
		_ = impl.SetFont("Roboto", "B", 9)
		ml, mt, mr, _ := impl.GetMargins()
		pw, _ := impl.GetPageSize()
		impl.GoPdf.SetXY(ml, mt/2)
		impl.SetTextColor(80, 80, 80)
		impl.CellFormat(pw-ml-mr, 14, "Header / Footer Sample", "B", 0, "L", false, 0, "")
	}
}

func gopdfPageNumber(impl *gopdf.Impl, _ fonts.Cache) func() {
	return func() {
		_ = impl.SetFont("Roboto", "", 8)
		ml, _, mr, mb := impl.GetMargins()
		pw, ph := impl.GetPageSize()
		impl.GoPdf.SetXY(ml, ph-(mb/2)-10)
		impl.SetTextColor(80, 80, 80)
		impl.CellFormat(pw-ml-mr, 10, fmt.Sprintf("Page %d", impl.GoPdf.GetNumberOfPages()),
			"T", 0, "C", false, 0, "")
	}
}

// backends in render order. Each factory takes the title (the .md basename)
// and the per-example config, and returns a pdf.Option that wires its
// backend into the renderer.
var backends = []struct {
	name   string
	option func(title string, cfg exampleConfig) pdf.Option
}{
	{
		name: "fpdf",
		option: func(title string, cfg exampleConfig) pdf.Option {
			return pdf.WithFpdf(context.Background(), fpdf.Config{
				Title:      title,
				HeaderFunc: cfg.fpdfHeader,
				FooterFunc: cfg.fpdfFooter,
			})
		},
	},
	{
		name: "gopdf",
		option: func(title string, cfg exampleConfig) pdf.Option {
			return pdf.WithPDF(gopdf.New(context.Background(), gopdf.Config{
				Title:      title,
				HeaderFunc: cfg.gopdfHeader,
				FooterFunc: cfg.gopdfFooter,
			}, nil))
		},
	},
}

func main() {
	matches, err := filepath.Glob("examples/*.md")
	if err != nil {
		log.Fatalf("glob examples: %v", err)
	}
	if len(matches) == 0 {
		log.Fatal("no examples/*.md found — run from the repo root")
	}

	for _, in := range matches {
		base := strings.TrimSuffix(filepath.Base(in), ".md")
		cfg := exampleConfigs[base] // zero value (all nil hooks) if absent
		for _, be := range backends {
			out := filepath.Join("examples", base+"."+be.name+".pdf")
			if err := renderExample(in, out, be.option(base, cfg)); err != nil {
				log.Fatalf("%s [%s]: %v", base, be.name, err)
			}
			log.Printf("%s [%s] → %s", base, be.name, out)
		}
	}
}

// renderExample reads markdown from `in`, renders it through the backend
// described by `backendOpt`, and writes the resulting PDF to `out`.
// extension.Table enables GFM-style pipe tables; WithEscapeHTML(false)
// keeps "&" from turning into "&amp;" in headings and body text (PDF
// isn't HTML).
func renderExample(in, out string, backendOpt pdf.Option) error {
	source, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("read sample: %w", err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
		goldmark.WithRenderer(pdf.New(
			backendOpt,
			pdf.WithEscapeHTML(false),
		)),
	)

	outFile, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	if err := md.Convert(source, outFile); err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	return nil
}
