package gopdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/go-swiss/fonts"
	"github.com/signintech/gopdf"
	"github.com/signintech/gopdf/fontmaker/core"
	pdf "github.com/stephenafamo/goldmark-pdf"
)

var (
	// Compile-time check that *Impl satisfies the pdf.PDF interface
	_ pdf.PDF = (*Impl)(nil)
)

// registeredImage caches the parsed holder and natural dimensions so UseImage
// can honor h=0 (auto-scale to aspect ratio) without re-decoding.
type registeredImage struct {
	holder        gopdf.ImageHolder
	naturalWidth  float64
	naturalHeight float64
}

// paperSizeFromConfig resolves the configured page size. PaperSizeRect wins
// when non-nil (escape hatch for arbitrary dimensions); otherwise the named
// PaperSize string is mapped to a gopdf preset. Empty or
// unrecognized strings fall back to A4 — matches fpdf.Config's "Default A4"
// behavior so the two backends behave identically on zero-config use.
func paperSizeFromConfig(c Config) gopdf.Rect {
	switch strings.ToUpper(c.PaperSize) {
	case "A3":
		return *gopdf.PageSizeA3
	case "A5":
		return *gopdf.PageSizeA5
	case "LETTER":
		return *gopdf.PageSizeLetter
	case "LEGAL":
		return *gopdf.PageSizeLegal
	case "TABLOID":
		return *gopdf.PageSizeTabloid
	default:
		return *gopdf.PageSizeA4
	}
}

func New(ctx context.Context, c Config, fontsCache fonts.Cache) *Impl {
	// gopdf has no orientation flag — pages are sized purely by W/H. Copy
	// the Rect (so unexported fields ride along) and swap dimensions for
	// landscape.
	paperRect := paperSizeFromConfig(c)
	// Match gofpdf: "l"/"landscape" (case-insensitive) selects landscape;
	// everything else is portrait.
	switch strings.ToLower(c.Orientation) {
	case "l", "landscape":
		paperRect.W, paperRect.H = paperRect.H, paperRect.W
	}

	gpdf := &Impl{
		GoPdf:  &gopdf.GoPdf{},
		Width:  paperRect.W,
		Height: paperRect.H,
		images: make(map[string]registeredImage),
	}

	gpdf.GoPdf.Start(gopdf.Config{
		Unit:     gopdf.UnitPT,
		PageSize: paperRect,
	})

	// Apply default margins via the public interface methods so the
	// wrapper's defaults flow through the same code path callers use when
	// overriding margins post-construction. Matches gofpdf's effective
	// defaults (1cm L/T/R + 2cm bottom auto-page-break trigger).
	gpdf.SetMarginLeft(defaultMarginLTR)
	gpdf.SetMarginTop(defaultMarginLTR)
	gpdf.SetMarginRight(defaultMarginLTR)
	gpdf.SetMarginBottom(defaultMarginBottom)

	gpdf.GoPdf.SetInfo(gopdf.PdfInfo{
		Title:   c.Title,
		Subject: c.Subject,
	})

	// Pre-register the logo so HeaderFunc/FooterFunc closures can pull it up
	// with impl.UseImage("logo", ...). Registering before AddPage means it's
	// available the first time the header runs (gopdf fires headerFunc from
	// inside AddPage).
	if c.Logo != nil {
		gpdf.RegisterImage("logo", c.LogoFormat, c.Logo)
	}

	// Wire header/footer before AddPage so they fire on page 1. gopdf invokes
	// both at page-open, matching gofpdf's — footer closures still need to absolutely
	// position themselves via SetY(-margin) if they want to sit at the bottom of the page.
	if c.HeaderFunc != nil {
		gpdf.GoPdf.AddHeader(c.HeaderFunc(gpdf, fontsCache))
	}

	if c.FooterFunc != nil {
		gpdf.GoPdf.AddFooter(c.FooterFunc(gpdf, fontsCache))
	}

	// Preload the default text font before page 1. gopdf has no
	// built-in core fonts, so without this, HeaderFunc/FooterFunc closures
	// firing on page 1's AddPage have nothing to SetFont — the renderer's
	// own addStyleFonts pass runs later, after Convert() starts. Loading
	// here mirrors fpdf's "Helvetica always available" UX. Failures
	// (network down + cold cache) are swallowed: page 1's hooks no-op,
	// page 2+ still pick up the font once the renderer loads its set.
	_ = pdf.AddFonts(ctx, gpdf, []pdf.Font{pdf.FontRoboto}, fontsCache)

	gpdf.AddPage()

	return gpdf
}

// Impl wraps gopdf and mirrors graphics-state values that gopdf
// doesn't keep on `gp.curr` and that don't auto-replay into new pages'
// content streams. AddPage re-issues these on every page; text-drawing ops
// re-issue fill color on the way out because gopdf emits a non-stroking-color
// (`rg`) inside BT/ET to set the text color, and `rg` is the *global* fill
// color in PDF — it leaks past ET and would otherwise turn the next fill
// rect black. This mirrors PDF's native q/Q idiom; the wrapper models it
// because gopdf doesn't expose raw q/Q emission.
//
// Font and text color are also mirrored — not because gopdf forgets them
// across pages (it doesn't), but because HeaderFunc/FooterFunc closures
// run *inside* AddPage and routinely override both. Without restoration,
// a page break that fires mid-paragraph leaves the renderer's next chunk
// rendered in the header's font (which is what bug screenshot #2 showed).
//
// Setters that mutate mirrored fields take pointer receivers so field writes
// survive past the call; the interface stores `*Impl`.
type Impl struct {
	Width  float64
	Height float64
	GoPdf  *gopdf.GoPdf

	images map[string]registeredImage

	fillR, fillG, fillB       uint8
	strokeR, strokeG, strokeB uint8
	lineWidth                 float64
	lineWidthSet              bool // distinguish "never set" (don't restore) from "set to 0"

	fontFamily string
	fontStyle  string
	fontSize   int
	fontSet    bool

	textR, textG, textB uint8
	textColorSet        bool

	// Per-font TypoAscender/UnitsPerEm ratio, captured at AddFont time. Used
	// by the text-draw path to mimic gofpdf's font-agnostic baseline placement
	// (baseline = cellTop + 0.5·h + 0.3·Size) on top of gopdf, which
	// otherwise reads each font's OS/2 metrics and lands the baseline at
	// cellTop + typoAscender·Size. See drawCellBaselineAligned for the shift.
	fontAscenderRatio map[fontKey]float64

	// addedFonts records family+style combos already registered via AddFont,
	// so callers (notably the renderer's addStyleFonts pass running after
	// gopdf.New's own preload) don't double-register.
	addedFonts map[fontKey]struct{}

	// errs buffers non-fatal failures from text/image/measure paths so the
	// caller sees them via Write() instead of getting a panic mid-render.
	// Matches gofpdf's swallow-at-draw-time, surface-at-output-time
	// semantics — keeps a flaky font or undecodable image from killing the
	// whole pipeline.
	errs []error
}

// recordErr buffers a non-fatal rendering error. Write() returns errors.Join
// of the accumulated errs so the caller can inspect what was dropped.
func (f *Impl) recordErr(err error) {
	if err == nil {
		return
	}
	f.errs = append(f.errs, err)
}

type fontKey struct {
	family string
	style  string
}

// maybeAddPage matches gofpdf's auto-page-break trigger: add a new page if
// drawing the next element of height `lineHeight` would extend past the
// bottom margin.
//
// Bails out when the cursor is already past the printable bottom — that's
// the signature of a caller drawing in the bottom margin on purpose
// (typically a FooterFunc closure positioning itself with SetXY). Without
// this guard, the footer's draw would trigger AddPage, which fires the
// footer closure again, which triggers another AddPage — infinite recursion
// until the goroutine stack blows.
func (f Impl) maybeAddPage(lineHeight float64) {
	Y := f.GoPdf.GetY()
	H := f.Height
	BottomM := f.GoPdf.MarginBottom()

	if Y > H-BottomM {
		return
	}

	if Y+lineHeight > H-BottomM {
		f.AddPage()
	}
}

// AddPage delegates to gopdf and re-issues every piece of wrapper-tracked
// graphics state onto the new page. PDF graphics state doesn't span pages
// (each content stream starts at the PDF defaults: black fill, black stroke,
// 1pt line width), and gopdf doesn't auto-replay these. Re-issuing also
// allocates the page's content stream — gopdf attaches /Contents lazily,
// and a page with no draw ops serializes with a malformed /Contents entry.
func (f Impl) AddPage() {
	f.GoPdf.AddPage()
	f.GoPdf.SetFillColor(f.fillR, f.fillG, f.fillB)
	f.GoPdf.SetStrokeColor(f.strokeR, f.strokeG, f.strokeB)
	if f.lineWidthSet {
		f.GoPdf.SetLineWidth(f.lineWidth)
	}
	// Font and text color restoration runs after fill/stroke so the page
	// looks visually identical to what the caller had before a HeaderFunc
	// or FooterFunc closure (fired by GoPdf.AddPage above) touched things.
	if f.fontSet {
		_ = f.GoPdf.SetFont(f.fontFamily, f.fontStyle, f.fontSize)
	}
	if f.textColorSet {
		f.GoPdf.SetTextColor(f.textR, f.textG, f.textB)
	}
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

// SetMarginBottom updates the bottom margin via the four-arg SetMargins call,
// preserving the current left/top/right margins. Drives the maybeAddPage
// auto-page-break trigger that fires when a draw op would cross
// `pageHeight - marginBottom`.
func (f Impl) SetMarginBottom(margin float64) {
	l, t, r, _ := f.GoPdf.Margins()
	f.GoPdf.SetMargins(l, t, r, margin)
}

func (f *Impl) SetFont(family string, style string, size int) error {
	f.fontFamily, f.fontStyle, f.fontSize, f.fontSet = family, style, size, true
	return f.GoPdf.SetFont(family, style, size)
}

// baselineShift returns the Y offset to apply before Top-aligned Cell draw so
// that the glyph baseline lands at the same font-agnostic position gofpdf uses:
//
//	baseline_y = cellTop + 0.5·h + 0.3·Size
//
// Why match gofpdf's formula: it puts the baseline at a fixed offset from the
// cell top regardless of font, so every font is visually centered in its line
// box AND runs across fonts (Roboto body next to Roboto Mono inline code)
// align without per-font compensation. gopdf's native behavior
// reads each font's OS/2 sTypoAscender and lands the baseline at
// cellTop + typoAscender·Size — that's font-specific, so it misaligns
// cross-font runs AND mis-centers within the line box when the font's
// metrics are tuned for tight web line-heights. Modern Google Roboto's
// sTypoAscender = 1536/2048 ≈ 0.75 (vs the older ~0.93) is the case that
// motivated this.
//
// Math: gopdf Top alignment puts baseline at curr.y + typoAscender·Size.
// We want baseline at cellTop + 0.5·h + 0.3·Size. So shift curr.y down by
// (0.5·h + 0.3·Size − typoAscender·Size) before the draw and restore after.
//
// If we couldn't parse the font (typoAscender unknown), shift is 0 — the
// result falls back to gofpdf native behavior for that font.
func (f Impl) baselineShift(h float64) float64 {
	if !f.fontSet {
		return 0
	}
	// Strip "U" before lookup: underline is a rendering flag, not a font
	// variant, so we register Roboto under "" / "B" / "I" / "BI" only.
	// gopdf's SetFont matches the same way (style&^Underline).
	style := strings.ReplaceAll(strings.ToUpper(f.fontStyle), "U", "")
	thisAsc, ok := f.fontAscenderRatio[fontKey{family: f.fontFamily, style: style}]
	if !ok {
		return 0
	}
	size := float64(f.fontSize)
	return 0.5*h + 0.3*size - thisAsc*size
}

// WriteText writes flowing text from the current cursor, breaking to the
// next line on each `\n` and wrapping each piece at word boundaries via
// WriteTextLine. Matches gofpdf.Write semantics.
//
// Pointer receiver so downstream draws can call recordErr through the
// same Impl when CellWithOption fails.
func (f *Impl) WriteText(h float64, text string) {
	if text == "" {
		return
	}
	leftMargin, _, _, _ := f.GoPdf.Margins()
	for i, segment := range strings.Split(text, "\n") {
		if i > 0 {
			f.GoPdf.Br(h)
			f.GoPdf.SetX(leftMargin)
		}
		f.WriteTextLine(h, segment)
	}
}

// WriteTextLine lays out a single logical line of text (no internal newlines)
// from the current cursor, wrapping at word boundaries when the next chunk
// would cross the right margin and continuing on subsequent visual lines from
// the left margin. A chunk wider than a full line falls back to a character
// break so progress is always made.
func (f *Impl) WriteTextLine(h float64, text string) {
	if strings.ContainsRune(text, '\n') {
		// Robustness: fall back to WriteText (which handles newlines) instead
		// of panicking on an internal contract violation.
		f.WriteText(h, text)
		return
	}

	if text == "" {
		return
	}

	pageWidth, _ := f.GetPageSize()
	leftMargin, _, rightMargin, _ := f.GoPdf.Margins()
	rightX := pageWidth - rightMargin

	draw := func(s string) {
		if s == "" {
			return
		}
		f.maybeAddPage(h)
		w, _ := f.GoPdf.MeasureTextWidth(s)
		f.drawCellBaselineAligned(w, h, s)
	}

	wrap := func() {
		f.GoPdf.Br(h)
		f.GoPdf.SetX(leftMargin)
	}

	// Chunks are alternating non-space / space runs so whitespace can be
	// dropped at wrap points (matching gofpdf, which doesn't carry leading
	// spaces onto new lines).
	for _, chunk := range splitWords(text) {
		availW := rightX - f.GoPdf.GetX()
		chunkW, _ := f.GoPdf.MeasureTextWidth(chunk)

		if chunkW <= availW {
			draw(chunk)
			continue
		}

		// Pure-whitespace chunk that doesn't fit: just wrap.
		if strings.TrimSpace(chunk) == "" {
			wrap()
			continue
		}

		// Word doesn't fit. Wrap first if anything's already on this line.
		if f.GoPdf.GetX() > leftMargin {
			wrap()
			availW = rightX - leftMargin
		}

		// Word still wider than a full line — break at character boundary
		// to make forward progress.
		if chunkW > availW {
			head, tail := breakAtWidth(f.GoPdf, chunk, availW)
			draw(head)
			for tail != "" {
				wrap()
				availW = rightX - leftMargin
				head, tail = breakAtWidth(f.GoPdf, tail, availW)
				draw(head)
			}
			continue
		}

		draw(chunk)
	}
}

func (f *Impl) CellFormat(w float64, h float64, txtStr string, borderStr string, ln int, alignStr string, fill bool, link int, linkStr string) {
	f.maybeAddPage(h)
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

	// Paint the background only when requested. The fill color is whatever
	// the renderer's SetStyle most recently set; the post-text fill pop in
	// the CellWithOption block below keeps it from leaking to black.
	if fill {
		x := f.GoPdf.GetX()
		y := f.GoPdf.GetY()
		f.GoPdf.RectFromUpperLeftWithStyle(x, y, w, h, "F")
	}

	// alignStr: horizontal (L/C/R, default L) + vertical (T/M/B, default M),
	// matching gofpdf's "LM"-as-default. Vertical-middle keeps table-cell
	// text visually consistent with the fpdf backend.
	upper := strings.ToUpper(alignStr)
	align := 0
	switch {
	case strings.Contains(upper, "C"):
		align |= gopdf.Center
	case strings.Contains(upper, "R"):
		align |= gopdf.Right
	default:
		align |= gopdf.Left
	}
	switch {
	case strings.Contains(upper, "T"):
		// Top is gopdf's default when no vertical flag is set.
	case strings.Contains(upper, "B"):
		align |= gopdf.Bottom
	default:
		align |= gopdf.Middle
	}

	err := f.GoPdf.CellWithOption(&gopdf.Rect{W: w, H: h}, txtStr, gopdf.CellOption{
		Align:  align,
		Border: border,
		Float:  float,
	})
	if err != nil {
		f.recordErr(fmt.Errorf("gopdf: CellFormat draw: %w", err))
	}
	// Pop the fill color the text op clobbered. Always run — gopdf may have
	// emitted partial PDF commands before the error, so the next op still
	// needs the correct fill state.
	f.GoPdf.SetFillColor(f.fillR, f.fillG, f.fillB)
}

func (f Impl) AddInternalLink(anchor string) {
	f.GoPdf.SetAnchor(anchor)
}

func (f *Impl) WriteInternalLink(h float64, text string, anchor string) {
	f.maybeAddPage(h)
	x := f.GoPdf.GetX()
	y := f.GoPdf.GetY()
	w, _ := f.GoPdf.MeasureTextWidth(text)
	f.drawCellBaselineAligned(w, h, text)
	// AddInternalLink uses the unshifted cell bounds: the link's clickable
	// region tracks the visual cell rect (which is anchored at the unshifted
	// y), even though the glyphs inside it shifted to align baselines.
	f.GoPdf.AddInternalLink(anchor, x, y, w, h)
}

func (f *Impl) WriteExternalLink(h float64, text string, destination string) {
	f.maybeAddPage(h)
	x := f.GoPdf.GetX()
	y := f.GoPdf.GetY()
	w, _ := f.GoPdf.MeasureTextWidth(text)
	f.drawCellBaselineAligned(w, h, text)
	f.GoPdf.AddExternalLink(destination, x, y, w, h)
}

// drawCellBaselineAligned draws a Cell at the current cursor with the
// font-agnostic gofpdf baseline placement applied via Y pre-shift (see
// baselineShift). Centralizes the save-y / shift / draw / restore-y
// dance so every text-draw path (WriteTextLine, WriteInternalLink,
// WriteExternalLink) shares it.
func (f *Impl) drawCellBaselineAligned(w, h float64, text string) {
	yShift := f.baselineShift(h)
	savedY := f.GoPdf.GetY()
	if yShift != 0 {
		f.GoPdf.SetY(savedY + yShift)
	}

	err := f.GoPdf.CellWithOption(&gopdf.Rect{W: w, H: h}, text, gopdf.CellOption{
		Align: gopdf.Left,
		Float: gopdf.Right,
	})

	if yShift != 0 {
		f.GoPdf.SetY(savedY)
	}

	if err != nil {
		f.recordErr(fmt.Errorf("gopdf: cell draw: %w", err))
	}
	// Pop the fill color the text op clobbered. Always run — see CellFormat.
	f.GoPdf.SetFillColor(f.fillR, f.fillG, f.fillB)
}

func (f Impl) BR(height float64) {
	f.maybeAddPage(height)
	f.GoPdf.Br(height)
}

// Images
func (f *Impl) RegisterImage(id string, format string, src io.Reader) {
	if f.images == nil {
		return
	}
	if _, ok := f.images[id]; ok {
		return
	}

	data, err := io.ReadAll(src)
	if err != nil {
		f.recordErr(fmt.Errorf("gopdf: read image %q: %w", id, err))
		return
	}

	holder, err := gopdf.ImageHolderByBytes(data)
	if err != nil {
		f.recordErr(fmt.Errorf("gopdf: decode image %q: %w", id, err))
		return
	}

	// Natural dimensions let UseImage honor the renderer's h=0 (auto-height)
	// contract. DecodeConfig only reads headers, so it's cheap.
	var naturalW, naturalH float64
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		naturalW = float64(cfg.Width)
		naturalH = float64(cfg.Height)
	}

	f.images[id] = registeredImage{
		holder:        holder,
		naturalWidth:  naturalW,
		naturalHeight: naturalH,
	}
}

func (f *Impl) UseImage(imgID string, x, y, w, h float64) {
	img, ok := f.images[imgID]
	if !ok {
		return
	}

	// The renderer passes h=0 to mean "auto-scale to aspect ratio".
	if h == 0 && img.naturalWidth > 0 && img.naturalHeight > 0 {
		h = w * img.naturalHeight / img.naturalWidth
	}

	f.maybeAddPage(h)

	rect := &gopdf.Rect{W: w, H: h}
	if err := f.GoPdf.ImageByHolderWithOptions(img.holder, gopdf.ImageOptions{
		X:    x,
		Y:    f.GoPdf.GetY(),
		Rect: rect,
	}); err != nil {
		f.recordErr(fmt.Errorf("gopdf: draw image %q: %w", imgID, err))
		return
	}

	// gopdf leaves Y put; advance it so subsequent content flows below the image.
	f.GoPdf.SetY(f.GoPdf.GetY() + h)
}

// Measuring
func (f *Impl) MeasureTextWidth(text string) float64 {
	width, err := f.GoPdf.MeasureTextWidth(text)
	if err != nil {
		f.recordErr(fmt.Errorf("gopdf: measure text width: %w", err))
		return 0
	}
	return width
}

func (f *Impl) SplitText(text string, width float64) []string {
	if text == "" {
		return nil
	}

	gp := f.GoPdf
	var lines []string
	var line []rune

	for _, r := range text {
		if r == '\n' {
			lines = append(lines, string(line))
			line = line[:0]
			continue
		}

		// Measure with the candidate rune included so the wrap point is the
		// rune that pushed us over — not the one after it.
		candidate := append(line, r) //nolint:gocritic // line is reassigned below; aliasing is intentional
		cw, err := gp.MeasureTextWidth(string(candidate))
		if err != nil {
			f.recordErr(fmt.Errorf("gopdf: measure for split: %w", err))
			// Best-effort fallback: include the rune without a width check so
			// we still make forward progress instead of dropping the tail.
			line = candidate
			continue
		}
		if cw > width && len(line) > 0 {
			lines = append(lines, string(line))
			line = []rune{r}
			continue
		}
		line = candidate
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return lines
}

// Colors
func (f *Impl) SetDrawColor(r uint8, g uint8, b uint8) {
	f.strokeR, f.strokeG, f.strokeB = r, g, b
	f.GoPdf.SetStrokeColor(r, g, b)
}

func (f *Impl) SetFillColor(r uint8, g uint8, b uint8) {
	f.fillR, f.fillG, f.fillB = r, g, b
	f.GoPdf.SetFillColor(r, g, b)
}

func (f *Impl) SetTextColor(r uint8, g uint8, b uint8) {
	f.textR, f.textG, f.textB, f.textColorSet = r, g, b, true
	f.GoPdf.SetTextColor(r, g, b)
}

// Width
func (f *Impl) SetLineWidth(width float64) {
	f.lineWidth, f.lineWidthSet = width, true
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

func (f *Impl) Write(w io.Writer) error {
	if _, err := f.GoPdf.WriteTo(w); err != nil {
		return err
	}
	// Surface any non-fatal errors collected during rendering. errors.Join
	// returns nil for an empty slice, so the happy path stays clean.
	return errors.Join(f.errs...)
}

func (f Impl) GetMargins() (left, top, right, bottom float64) {
	return f.GoPdf.Margins()
}

func (f *Impl) AddFont(family string, styleStr string, data []byte) error {
	styleStr = strings.ToUpper(styleStr)
	key := fontKey{family: family, style: styleStr}

	// Skip work if this family+style was already registered — ascender ratio
	// is already cached and gopdf would otherwise leak duplicate
	// font objects into the output PDF (see addedFonts comment on Impl).
	if _, ok := f.addedFonts[key]; ok {
		return nil
	}

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

	// Capture this font's TypoAscender/UnitsPerEm ratio so the text-draw path
	// can apply gofpdf-style baseline placement (see baselineShift).
	// Parser errors are non-fatal: fall through and let gopdf
	// surface them on AddTTFFontDataWithOption; the draw path then falls
	// back to native baseline for that font.
	parser := &core.TTFParser{}
	if err := parser.ParseByReader(bytes.NewReader(data)); err == nil {
		upem := parser.UnitsPerEm()
		if upem > 0 {
			if f.fontAscenderRatio == nil {
				f.fontAscenderRatio = make(map[fontKey]float64)
			}
			f.fontAscenderRatio[key] = float64(parser.TypoAscender()) / float64(upem)
		}
	}

	if err := f.GoPdf.AddTTFFontDataWithOption(family, data, gopdf.TtfOption{
		Style: style,
	}); err != nil {
		return err
	}

	if f.addedFonts == nil {
		f.addedFonts = make(map[fontKey]struct{})
	}
	f.addedFonts[key] = struct{}{}
	return nil
}
