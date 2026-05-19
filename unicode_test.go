package pdf

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Regression test for https://github.com/stephenafamo/goldmark-pdf/issues/27.
// Non-BMP runes (here U+1F6E1 🛡) used to crash gofpdf's generateCIDFontMap
// because its Cw array is fixed at 65536 entries.
//
// The crash only reproduces against a real UTF-8 font added via
// AddUTF8FontFromBytes — inbuilt PDF core fonts (Helvetica/Times/Courier)
// skip the CID-map path entirely. Default config uses Google Roboto, which
// addStyleFonts fetches over the network, so this test is gated on -short.
func TestFpdf_RenderNonBMPCharInCodeBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: downloads Google fonts over the network")
	}

	const sample = "## Einzelnachweise\n\nTest\n\n```default\n🛡\n```\n"

	md := goldmark.New(
		goldmark.WithRenderer(New(
			WithFpdf(context.Background(), FpdfConfig{Title: "issue-27"}),
		)),
		goldmark.WithExtensions(extension.GFM, extension.Table),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(sample), &buf); err != nil {
		t.Fatalf("convert: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}

	// Sanity: a successful render produced bytes; drop them.
	_, _ = io.Copy(io.Discard, &buf)
}
