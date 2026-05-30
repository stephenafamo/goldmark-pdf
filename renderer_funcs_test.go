package pdf

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// renderText is registered for both KindText and KindString but
// unconditionally cast to *ast.Text, panicking when an extension emits
// *ast.String (e.g. Typographer's smart-quote/dash substitutions).
func TestRender_AstStringDoesNotPanic(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Typographer),
		goldmark.WithRenderer(New(
			WithPDF(&MockPdf{pageWidth: 600, pageHeight: 800, leftMargin: 50, rightMargin: 50}),
			// Inbuilt fonts skip the Google-fonts network fetch.
			WithHeadingFont(FontHelvetica),
			WithBodyFont(FontHelvetica),
			WithCodeFont(FontCourier),
		)),
	)

	// Typographer rewrites the dashes, ellipsis, and quotes into *ast.String
	// nodes interleaved with the surrounding *ast.Text.
	const sample = "Hello -- world... and \"quotes\".\n"

	var buf bytes.Buffer
	if err := md.Convert([]byte(sample), &buf); err != nil {
		t.Fatalf("convert: %v", err)
	}
}
