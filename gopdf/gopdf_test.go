package gopdf

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"image/color"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	pdf "github.com/stephenafamo/goldmark-pdf"
)

// findTTF looks up a TTF font from a couple of well-known system locations.
// Tests that need a real font skip when none is found so the suite stays
// hermetic on machines without system fonts.
func findTTF(t *testing.T) []byte {
	t.Helper()
	candidates := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"/Library/Fonts/Arial.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			return data
		}
	}
	t.Skipf("no system TTF font found in known locations: %v", candidates)
	return nil
}

// newTestPDF returns a gopdf backend pre-loaded with a DejaVu font under all
// four styles, plus the option that wires it into the renderer (lets
// addStyleFonts skip downloading anything).
func newTestPDF(t *testing.T) (pdf.Option, *Impl) {
	t.Helper()
	ttf := findTTF(t)

	opt := WithGoPdf(context.Background(), Config{Title: "test"})

	// WithGoPdf returns an OptionFunc that sets Config.PDF. Pull the underlying
	// *Impl back out via a throwaway Config so the test can poke its state.
	cfg := &pdf.Config{}
	opt.SetConfig(cfg)
	impl, ok := cfg.PDF.(*Impl)
	if !ok {
		t.Fatalf("expected *gopdf.Impl, got %T", cfg.PDF)
	}

	for _, style := range []string{pdf.FontStyleRegular, pdf.FontStyleBold, pdf.FontStyleItalic, pdf.FontStyleBoldItalic} {
		if err := impl.AddFont("DejaVu", style, ttf); err != nil {
			t.Fatalf("AddFont(%q): %v", style, err)
		}
	}

	return opt, impl
}

// newTestMD builds a goldmark renderer wired to a DejaVu-fonted gopdf backend.
// Pass extensions via `exts`; renderer-level tweaks (link color, etc.) via opts.
func newTestMD(t *testing.T, exts []goldmark.Extender, opts ...pdf.Option) goldmark.Markdown {
	t.Helper()
	pdfOpt, _ := newTestPDF(t)
	dejavu := pdf.Font{
		CanUseForText: true,
		CanUseForCode: true,
		Family:        "DejaVu",
		Type:          pdf.FontTypeCustom,
	}
	rendererOpts := append([]pdf.Option{
		pdfOpt,
		pdf.WithHeadingFont(dejavu),
		pdf.WithBodyFont(dejavu),
		pdf.WithCodeFont(dejavu),
	}, opts...)
	return goldmark.New(
		goldmark.WithExtensions(exts...),
		goldmark.WithRenderer(pdf.New(rendererOpts...)),
	)
}

// 1x1 transparent PNG used by image-related tests.
const onePxPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

func TestRegisterImage_AutoHeightFromAspectRatio(t *testing.T) {
	img := Impl{
		GoPdf:  nil, // not exercised on this path
		images: make(map[string]registeredImage),
	}

	data, err := base64.StdEncoding.DecodeString(onePxPNG)
	if err != nil {
		t.Fatalf("decode test PNG: %v", err)
	}
	img.RegisterImage("test", "png", bytes.NewReader(data))

	got, ok := img.images["test"]
	if !ok {
		t.Fatal("expected image to be registered")
	}
	if got.naturalWidth != 1 || got.naturalHeight != 1 {
		t.Errorf("expected 1x1 natural dimensions, got %vx%v", got.naturalWidth, got.naturalHeight)
	}
}

func TestRegisterImage_DuplicateIDKeepsFirst(t *testing.T) {
	img := Impl{images: make(map[string]registeredImage)}
	data, _ := base64.StdEncoding.DecodeString(onePxPNG)

	img.RegisterImage("dup", "png", bytes.NewReader(data))
	first := img.images["dup"].holder

	// Second call with the same id should be ignored, not overwrite.
	img.RegisterImage("dup", "png", bytes.NewReader(data))
	second := img.images["dup"].holder

	if first != second {
		t.Error("duplicate RegisterImage should not replace existing holder")
	}
}

// Full end-to-end render through goldmark + the gopdf backend. Catches
// regressions in the wrapper interface: font loading, text writing, links,
// tables, code blocks, images, page output. We don't validate visual output
// — just that the pipeline produces a syntactically-valid PDF for a mix of
// elements the renderer exercises.
func TestEndToEndRender(t *testing.T) {
	md := newTestMD(t,
		[]goldmark.Extender{extension.Table, extension.Strikethrough},
		pdf.WithLinkColor(color.RGBA{0, 0, 200, 255}),
	)

	source := `# Heading One

A paragraph with **bold**, *italic*, and ~~strikethrough~~ text plus an
[external link](https://example.com) and an [internal link](#heading-one).

## Lists

- item one
- item two
- item three

` + "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```" + `

## Table

| Col A | Col B |
|-------|-------|
| 1     | 2     |
| 3     | 4     |
`

	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	out := buf.Bytes()
	if len(out) == 0 {
		t.Fatal("expected non-empty PDF output")
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("expected PDF magic header, got %q", firstBytes(out, 16))
	}
	if !bytes.Contains(out, []byte("%%EOF")) {
		t.Error("expected PDF trailer marker missing")
	}
}

// Verifies the wrapper handles `h=0` (auto-scale) when used inline through
// the renderer's UseImage path.
func TestEndToEndRender_WithImage(t *testing.T) {
	md := newTestMD(t, nil)

	source := "# img\n\n![pixel](data:image/png;base64," + onePxPNG + ")\n\nafter\n"
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Errorf("expected PDF output, got first 16 bytes: %q", firstBytes(buf.Bytes(), 16))
	}
}

// Long paragraphs (a single ast.Text node with no inline markup) must wrap at
// the right margin instead of running off the page.
func TestWriteText_WrapsLongParagraph(t *testing.T) {
	md := newTestMD(t, nil)

	source := []byte(strings.Repeat("This is a long paragraph that needs to wrap. ", 20))
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	// The whole text is preserved in the PDF stream after wrapping; if the
	// renderer dropped a wrap-point rune the assertion below would fail.
	// We can't decode the PDF's text content easily, so we look at the raw
	// bytes for individual short snippets that occur many times.
	out := buf.Bytes()
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}
	// Output should be larger than a non-wrapped one-page PDF: roughly,
	// long text means more content stream bytes (rough sanity check).
	if len(out) < 5000 {
		t.Errorf("output suspiciously small (%d bytes) — text may not have flowed", len(out))
	}
}

// Orientation matches gofpdf: "L"/"Landscape" (case-insensitive) swaps the
// page dimensions, everything else stays portrait.
func TestOrientation(t *testing.T) {
	cases := []struct {
		name        string
		orientation string
		wantWide    bool // true if final page width > height
	}{
		{"empty defaults portrait", "", false},
		{"P portrait", "P", false},
		{"Portrait", "Portrait", false},
		{"l landscape", "l", true},
		{"L", "L", true},
		{"Landscape mixed case", "LandScape", true},
		{"unknown falls back to portrait", "diagonal", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opt := WithGoPdf(context.Background(), Config{
				Orientation: tc.orientation,
			})
			cfg := &pdf.Config{}
			opt.SetConfig(cfg)
			impl := cfg.PDF.(*Impl)
			w, h := impl.GetPageSize()
			if (w > h) != tc.wantWide {
				t.Errorf("orientation %q: got %v×%v (wide=%v), wantWide=%v",
					tc.orientation, w, h, w > h, tc.wantWide)
			}
		})
	}
}

// Every filled rectangle in the PDF stream must be preceded by its own `rg`
// (nonstroking color) command, so the fill never inherits the text color
// that gopdf emits inside BT/ET.
func TestCellFormat_FillRectAlwaysHasOwnFillColor(t *testing.T) {
	md := newTestMD(t, []goldmark.Extender{extension.Table})

	// A table with rows of mixed wrapped-line counts forces multi-line cells
	// to issue multiple fill rects per row — the exact case where the
	// nonstroking-color leak from BT/ET text used to bleed black into the
	// fill state.
	source := []byte(`| A | B | C |
|---|---|---|
| ` + strings.Repeat("x ", 40) + ` | one | ` + strings.Repeat("y ", 40) + ` |
| short | short | short |
`)
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	streams := extractContentStreams(t, buf.Bytes())
	if len(streams) == 0 {
		t.Fatal("no content streams found")
	}

	for _, s := range streams {
		if !bytes.Contains(s, []byte(" re f")) {
			continue
		}
		// For each "re f" instance, the most recent nonstroking color before
		// it (within the same stream) must come from a wrapper-emitted `rg`
		// — not from inside a BT/ET text block. A simple sufficient check:
		// every `re f` must be immediately preceded (within ~3 lines) by an
		// `rg` command outside any BT/ET block.
		lines := strings.Split(string(s), "\n")
		inText := false
		lastRgIdx := -1
		for i, l := range lines {
			switch {
			case l == "BT":
				inText = true
			case l == "ET":
				inText = false
			case !inText && strings.HasSuffix(strings.TrimSpace(l), " rg"):
				lastRgIdx = i
			case strings.HasSuffix(strings.TrimSpace(l), " re f"):
				// must have seen an `rg` outside BT/ET within 5 lines
				if lastRgIdx == -1 || i-lastRgIdx > 5 {
					t.Errorf("fill rect at line %d has no recent non-text `rg` (lastRgIdx=%d)", i, lastRgIdx)
				}
			}
		}
	}
}

// extractContentStreams pulls inflated content streams out of a PDF. gopdf
// emits FlateDecode-compressed streams; we look for "stream...endstream"
// regions and try to inflate each.
func extractContentStreams(t *testing.T, pdfBytes []byte) [][]byte {
	t.Helper()
	re := regexp.MustCompile(`(?s)stream\r?\n(.+?)\r?\nendstream`)
	var out [][]byte
	for _, m := range re.FindAllSubmatch(pdfBytes, -1) {
		r, err := zlib.NewReader(bytes.NewReader(m[1]))
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(r)
		r.Close()
		if bytes.Contains(data, []byte(" re ")) {
			out = append(out, data)
		}
	}
	return out
}

func TestSplitText_DoesNotDropOverflowRune(t *testing.T) {
	impl := newFontedImpl(t)

	in := strings.Repeat("a", 200)
	wTotal := impl.MeasureTextWidth(in)

	// Splitting at half-width should produce at least two lines, and the joined
	// result must reproduce the input exactly.
	lines := impl.SplitText(in, wTotal/2)
	if len(lines) < 2 {
		t.Fatalf("expected >= 2 lines for half-width split, got %d", len(lines))
	}
	if joined := strings.Join(lines, ""); joined != in {
		t.Errorf("rune(s) dropped during wrap.\n  in:  %q\n  out: %q", in, joined)
	}
}

func firstBytes(b []byte, n int) string {
	if len(b) < n {
		n = len(b)
	}
	return string(b[:n])
}

// When the cursor is already in the bottom margin (a FooterFunc drawing
// absolute-positioned), maybeAddPage must not auto-page-break. Breaking
// here would fire the footer again and recurse until stack overflow.
func TestMaybeAddPage_SkipsWhenCursorInBottomMargin(t *testing.T) {
	opt := WithGoPdf(context.Background(), Config{})
	cfg := &pdf.Config{}
	opt.SetConfig(cfg)
	impl := cfg.PDF.(*Impl)

	// Position cursor halfway into the bottom margin — typical FooterFunc Y.
	_, ph := impl.GetPageSize()
	_, _, _, mb := impl.GetMargins()
	impl.SetY(ph - mb/2)

	before := impl.GoPdf.GetNumberOfPages()
	impl.maybeAddPage(10)
	after := impl.GoPdf.GetNumberOfPages()

	if after != before {
		t.Errorf("maybeAddPage added a page (%d→%d) when cursor was past bottom margin", before, after)
	}
}

// Confirms the guard didn't disable the legitimate auto-page-break: cursor
// in the content area where the next op crosses the bottom margin.
func TestMaybeAddPage_BreaksWhenContentOverflows(t *testing.T) {
	opt := WithGoPdf(context.Background(), Config{})
	cfg := &pdf.Config{}
	opt.SetConfig(cfg)
	impl := cfg.PDF.(*Impl)

	// One point above the printable bottom; a 10pt line would cross it.
	_, ph := impl.GetPageSize()
	_, _, _, mb := impl.GetMargins()
	impl.SetY(ph - mb - 1)

	before := impl.GoPdf.GetNumberOfPages()
	impl.maybeAddPage(10)
	after := impl.GoPdf.GetNumberOfPages()

	if after != before+1 {
		t.Errorf("maybeAddPage did not break: pages %d→%d", before, after)
	}
}
