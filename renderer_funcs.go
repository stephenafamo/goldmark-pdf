package pdf

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	textm "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// A NodeRenderer interface offers NodeRendererFuncs.
type NodeRenderer interface {
	// RendererFuncs registers NodeRendererFuncs to given NodeRendererFuncRegisterer.
	RegisterFuncs(NodeRendererFuncRegisterer)
}

// A NodeRendererFuncRegisterer registers
type NodeRendererFuncRegisterer interface {
	// Register registers given NodeRendererFunc to this object.
	Register(ast.NodeKind, NodeRendererFunc)
}

// A function to render an ast.Node to the given Writer. The writer contains the PDF and style information
type NodeRendererFunc func(w *Writer, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error)

type nodeRederFuncs struct{}

// RegisterFuncs implements NodeRenderer.RegisterFuncs .
func (r *nodeRederFuncs) RegisterFuncs(reg NodeRendererFuncRegisterer) {
	// blocks
	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindTextBlock, r.renderTextBlock)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)

	// inlines
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindLink, r.renderLink)
	// m[ast.KindRawHTML] = r.renderRawHTML // Not applicable to PDF
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderText)

	// GFM Extensions
	// Tables
	reg.Register(east.KindTable, r.renderTable)
	reg.Register(east.KindTableHeader, r.renderTableHeader)
	reg.Register(east.KindTableRow, r.renderTableRow)
	reg.Register(east.KindTableCell, r.renderTableCell)
	// Strikethrough
	reg.Register(east.KindStrikethrough, r.renderStrikethrough)
	// Checkbox
	reg.Register(east.KindTaskCheckBox, r.renderTaskCheckBox)
}

func (r *nodeRederFuncs) renderDocument(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Nothing to do
	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderHeading(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		w.Pdf.BR(w.States.peek().textStyle.Size + (w.States.peek().textStyle.Spacing * 2))
		id, _ := n.AttributeString("id")
		if anchor, ok := id.([]byte); ok {
			w.Pdf.AddInternalLink(string(anchor))
		}

		x := &state{
			containerType: n.Kind(),
			listkind:      notlist,
			leftMargin:    w.States.peek().leftMargin,
		}

		switch n.Level {
		case 1:
			w.LogDebug("Heading (1, entering)", "")
			x.textStyle = *w.Styles.H1
		case 2:
			w.LogDebug("Heading (2, entering)", "")
			x.textStyle = *w.Styles.H2
		case 3:
			w.LogDebug("Heading (3, entering)", "")
			x.textStyle = *w.Styles.H3
		case 4:
			w.LogDebug("Heading (4, entering)", "")
			x.textStyle = *w.Styles.H4
		case 5:
			w.LogDebug("Heading (5, entering)", "")
			x.textStyle = *w.Styles.H5
		case 6:
			w.LogDebug("Heading (6, entering)", "")
			x.textStyle = *w.Styles.H6
		}

		w.States.push(x)
	} else {
		w.LogDebug("Heading (leaving)", "")
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
		w.States.pop()
	}
	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderTextBlock(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		w.LogDebug("Text Block", "")
		if node.FirstChild() != nil {
			w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
		}
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderText(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Text)
	segment := n.Segment
	w.WriteText(string(segment.Value(source)))

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderBlockquote(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	SetStyle(w.Pdf, *w.Styles.Blockquote)
	indent := w.Pdf.MeasureTextWidth("m") * 3

	if entering {
		w.LogDebug("BlockQuote (entering)", "")
		curleftmargin, _, _, _ := w.Pdf.GetMargins()
		x := &state{
			containerType: ast.KindBlockquote,
			textStyle:     *w.Styles.Blockquote, listkind: notlist,
			leftMargin: curleftmargin + indent,
		}
		w.States.push(x)
		w.Pdf.SetMarginLeft(curleftmargin + indent)
	} else {
		w.LogDebug("BlockQuote (leaving)", "")
		curleftmargin, _, _, _ := w.Pdf.GetMargins()
		w.Pdf.SetMarginLeft(curleftmargin - indent)
		w.States.pop()
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderCodeBlock(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		w.LogDebug("Codeblock Leaving", "")
		return ast.WalkContinue, nil
	}

	mleft, _, mright, _ := w.Pdf.GetMargins()

	n := node.(interface {
		Lines() *textm.Segments
	})
	w.LogDebug("Codeblock", "")

	var content string
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		content += string(line.Value(source))
	}
	content = strings.ReplaceAll(content, "\t", "    ")

	bgStyle := w.Styles.CodeBlockTheme.Get(chroma.Text)
	fBgStyle := w.ChromaToStyle(bgStyle)
	SetStyle(w.Pdf, *fBgStyle)
	w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing) // start on next line!

	lang := ""

	withLang, hasLang := node.(*ast.FencedCodeBlock)
	if hasLang {
		lang = string(withLang.Language(source))
	}

	var lexer chroma.Lexer
	if lang == "" {
		lexer = lexers.Analyse(content)
	} else {
		lexer = lexers.Get(lang)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return ast.WalkStop, fmt.Errorf("could not tokenise fenced code block: %w", err)
	}

	height := fBgStyle.Size + fBgStyle.Spacing

	w.Pdf.SetMarginLeft(mleft + height)
	w.Pdf.SetMarginRight(mright + height)

	pageWidth, _ := w.Pdf.GetPageSize()

	addLeftPad := func() {
		prevX := w.Pdf.GetX()
		w.Pdf.SetX(mleft)
		w.Pdf.CellFormat((height), height, "", "TBL", 0, "L", true, 0, "")
		w.Pdf.SetX(prevX)
	}
	addRightPad := func() {
		X := w.Pdf.GetX()
		border := "TBR"
		if X == mleft+height {
			border = "1"
		}
		w.Pdf.CellFormat(pageWidth-X-(mright), height, "", border, 0, "L", true, 0, "")
		w.Pdf.BR(fBgStyle.Size + fBgStyle.Spacing)
	}

	// Add a padding before the code block
	w.Pdf.SetX(mleft)
	w.Pdf.CellFormat(pageWidth-mleft-mright, height, "", "1", 0, "L", true, 0, "")
	w.Pdf.BR(fBgStyle.Size + fBgStyle.Spacing)

	err = chroma.FormatterFunc(func(iow io.Writer, s *chroma.Style, iterator chroma.Iterator) error {
		for t := iterator(); t != chroma.EOF; t = iterator() {
			chSt := w.Styles.CodeBlockTheme.Get(t.Type)
			tokenStyle := w.ChromaToStyle(chSt)
			tokenHeight := tokenStyle.Size + tokenStyle.Spacing

			SetStyle(w.Pdf, *tokenStyle)

			tokenLines := strings.Split(t.Value, "\n")
			for k, v := range tokenLines {
				if k > 0 {
					addRightPad()
				}

				// Fill the left gutter on every row about to receive content.
				// X == mleft+height means a fresh row (just had a BR), so the
				// area mleft..mleft+height hasn't been filled yet — without
				// this, the first row of the block and any row reached via
				// the last line of a multi-line token would leak the page bg.
				if w.Pdf.GetX() == mleft+height {
					addLeftPad()
				}

				border := "TBR"
				if w.Pdf.GetX() == mleft+height {
					border = "1"
				}

				if v != "" {
					allChunks := w.Pdf.SplitText(v, pageWidth-mright-tokenHeight-w.Pdf.GetX())
					txt := allChunks[0]

					// Print the first chunk. Should be the only chunk if we do not
					// need to line break
					w.Pdf.CellFormat(
						w.Pdf.MeasureTextWidth(txt),
						tokenHeight, txt, border, 0, "L", true, 0, "")

					// If there were more chunks, we join the rest and split them by the
					// page width
					if len(allChunks) > 0 {
						for _, vn := range w.Pdf.SplitText(strings.Join(allChunks[1:], ""), pageWidth) {

							addRightPad()
							addLeftPad()

							txt := vn
							w.Pdf.CellFormat(w.Pdf.MeasureTextWidth(txt), tokenHeight, txt, "1", 0, "L", true, 0, "")
						}
					}
				}
			}
		}

		return nil
	}).Format(nil, w.Styles.CodeBlockTheme, iterator)
	if err != nil {
		return ast.WalkStop, fmt.Errorf("could not format fenced code block: %w", err)
	}

	w.Pdf.SetX(mleft)
	w.Pdf.CellFormat(pageWidth-mright-mleft, height, "", "1", 0, "L", true, 0, "")
	w.Pdf.BR(fBgStyle.Size + fBgStyle.Spacing)

	w.Pdf.SetMarginLeft(mleft)
	w.Pdf.SetMarginRight(mright)
	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderHTMLBlock(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Cannot process HTML blocks

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderList(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.List)
	kind := unordered
	if n.IsOrdered() {
		kind = ordered
	}

	LH := w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing

	SetStyle(w.Pdf, *w.Styles.Normal)
	if entering {
		w.LogDebug(fmt.Sprintf("%v List (entering)", kind), "")
		w.Pdf.SetMarginLeft(w.States.peek().leftMargin)
		w.LogDebug("... List Left Margin", fmt.Sprintf("set to %v", w.States.peek().leftMargin))

		x := &state{
			containerType: ast.KindList,
			textStyle:     *w.Styles.Normal, itemNumber: 0,
			listkind:   kind,
			leftMargin: w.States.peek().leftMargin,
		}

		w.Pdf.BR(LH)

		w.States.push(x)
	} else {
		w.LogDebug(fmt.Sprintf("%v List (leaving)", kind), "")
		w.Pdf.SetMarginLeft(w.States.peek().leftMargin)
		w.LogDebug("... Reset List Left Margin",
			fmt.Sprintf("re-set to %v", w.States.peek().leftMargin))
		w.States.pop()
		if len(w.States.stack) < 2 {
			w.Pdf.BR(LH)
		}
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderListItem(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		em := w.Pdf.MeasureTextWidth("m") * 2

		w.LogDebug(fmt.Sprintf("%v Item (entering) #%v", w.States.peek().listkind, w.States.peek().itemNumber+1), "")

		// with the bullet done, now set the left margin for the text
		X := w.States.peek().leftMargin + em
		// set the cursor to this point
		w.Pdf.SetX(X)

		x := &state{
			containerType: ast.KindListItem,
			textStyle:     *w.Styles.Normal, itemNumber: w.States.peek().itemNumber + 1,
			listkind:       w.States.peek().listkind,
			firstParagraph: true,
			leftMargin:     X,
		}
		w.States.push(x)

		// add bullet or itemnumber; then set left margin for the
		// text/paragraphs in the item
		SetStyle(w.Pdf, w.States.peek().textStyle)
		if w.States.peek().listkind == unordered {
			w.Pdf.CellFormat(em, w.Styles.Normal.Size+w.Styles.Normal.Spacing,
				"-",
				"", 0, "L", false, 0, "")
		} else if w.States.peek().listkind == ordered {
			w.Pdf.CellFormat(em, w.Styles.Normal.Size+w.Styles.Normal.Spacing,
				fmt.Sprintf("%v.", w.States.peek().itemNumber),
				"", 0, "L", false, 0, "")
		}

		w.Pdf.SetMarginLeft(w.States.peek().leftMargin + em)

	} else {
		w.LogDebug(fmt.Sprintf("%v Item (leaving)", w.States.peek().listkind), "")
		// before we output the new line, reset left margin
		w.Pdf.SetMarginLeft(w.States.peek().leftMargin)
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
		w.States.parent().itemNumber++
		w.States.pop()
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderParagraph(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	SetStyle(w.Pdf, *w.Styles.Normal)
	if entering {
		w.LogDebug("Paragraph (entering)", "")
		lm, tm, rm, bm := w.Pdf.GetMargins()
		w.LogDebug("... Margins (left, top, right, bottom:", fmt.Sprintf("%v %v %v %v", lm, tm, rm, bm))
		if w.States.peek().containerType == ast.KindListItem {
			t := w.States.peek().listkind
			if t == unordered || t == ordered || t == definition {
				if w.States.peek().firstParagraph {
					w.LogDebug("First Para within a list", "breaking")
				} else {
					w.LogDebug("Not First Para within a list", "indent etc.")
					w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
				}
			}
			return ast.WalkContinue, nil
		}
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
		// w.cr()
	} else {
		w.LogDebug("Paragraph (leaving)", "")
		lm, tm, rm, bm := w.Pdf.GetMargins()
		w.LogDebug("... Margins (left, top, right, bottom:",
			fmt.Sprintf("%v %v %v %v", lm, tm, rm, bm))
		if w.States.peek().containerType == ast.KindListItem {
			t := w.States.peek().listkind
			if t == unordered || t == ordered || t == definition {
				if w.States.peek().firstParagraph {
					w.States.peek().firstParagraph = false
				} else {
					w.LogDebug("Not First Para within a list", "")
					w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
				}
			}
			return ast.WalkContinue, nil
		}
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderThematicBreak(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	w.LogDebug("HorizontalRule", "")

	LH := w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing
	lineWidth := 3.0

	// do a newline
	w.Pdf.BR(LH)

	// get the page margins
	_, _, rm, _ := w.Pdf.GetMargins()
	// get the page size
	width, _ := w.Pdf.GetPageSize()

	// get the current x and y (assume left margin in ok)
	x := w.Pdf.GetX()

	// Center the rule in the next line of whitespace so the gap above
	// and below it is roughly equal.
	y := w.Pdf.GetY() + LH/2 + lineWidth/2

	// now compute the x value of the right side of page
	newx := width - rm

	w.Pdf.SetLineWidth(lineWidth)
	w.Pdf.SetDrawColor(200, 200, 200) // Line() strokes with the draw color.

	w.LogDebug("... From X,Y", fmt.Sprintf("%v,%v", x, y))
	w.Pdf.Line(x, y, newx, y)
	w.LogDebug("...   To X,Y", fmt.Sprintf("%v,%v", newx, y))

	// another newline
	w.Pdf.BR(LH)

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderLink(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		x := &state{
			containerType: ast.KindLink,
			textStyle:     *(w.GetLinkStyle()),
			listkind:      notlist,
			leftMargin:    w.States.peek().leftMargin,
			destination:   string(n.Destination),
		}
		w.States.push(x)
		w.LogDebug("Link (entering)", fmt.Sprintf("Destination[%v] Title[%v]", string(n.Destination), string(n.Title)))
	} else {
		w.LogDebug("Link (leaving)", "")
		w.States.pop()
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderAutoLink(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.AutoLink)
	if !entering {
		return ast.WalkContinue, nil
	}

	url := n.URL(source)
	label := n.Label(source)

	destination := ""
	if n.AutoLinkType == ast.AutoLinkEmail && !bytes.HasPrefix(bytes.ToLower(url), []byte("mailto:")) {
		destination += "mailto:"
	}
	destination += string(util.EscapeHTML(util.URLEscape(url, false)))

	w.LogDebug("AutoLink", fmt.Sprintf("Destination[%v] Title[%v]", destination, string(label)))

	x := &state{
		containerType: ast.KindAutoLink,
		textStyle:     *(w.GetLinkStyle()),
		listkind:      notlist,
		leftMargin:    w.States.peek().leftMargin,
		destination:   destination,
	}
	w.States.push(x)
	w.WriteText(string(label))
	w.States.pop()

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderCodeSpan(w *Writer, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	// maintain the container type so we can draw links
	if entering {
		w.LogDebug("Code (entering)", "")
		x := &state{
			containerType: ast.KindCodeSpan,
			textStyle:     *(w.GetBacktickStyle()),
			listkind:      notlist,
			leftMargin:    w.States.peek().leftMargin,
			destination:   w.States.peek().destination,
		}
		w.States.push(x)
	} else {
		w.LogDebug("Code (leaving)", "")
		w.States.pop()
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderImage(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// while this has entering and leaving states, it doesn't appear
	// to be useful except for other markup languages to close the tag
	n := node.(*ast.Image)

	if entering {
		w.LogDebug("Image (entering)", fmt.Sprintf("Destination[%v] Title[%v]", string(n.Destination), string(n.Title)))
		// following changes suggested by @sirnewton01, issue #6
		// does file exist?
		imgPath := localPath(string(n.Destination))
		imgFile, err := w.ImageFS.Open(imgPath)
		if err == nil {
			defer imgFile.Close()

			width, _ := w.Pdf.GetPageSize()
			mleft, _, mright, _ := w.Pdf.GetMargins()
			maxw := width - (mleft * 2) - (mright * 2)

			mimeType := getFileMime(imgFile)
			w.Pdf.RegisterImage(imgPath, getImageMime(mimeType), imgFile)
			w.Pdf.UseImage(imgPath, mleft*2, w.Pdf.GetY(), maxw, 0)
		} else {
			log.Printf("IMAGE ERROR: %s, %v", imgPath, err)
			w.LogDebug("Image (file error)", err.Error())
		}
	} else {
		w.LogDebug("Image (leaving)", "")
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderEmphasis(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Emphasis)
	if entering {
		switch n.Level {
		case 2:
			w.LogDebug("Strong (entering)", "")
			w.States.peek().textStyle.format = strings.ReplaceAll(w.States.peek().textStyle.format, "B", "")
			w.States.peek().textStyle.format += "B"
		default:
			w.LogDebug("Emph (entering)", "")
			w.States.peek().textStyle.format = strings.ReplaceAll(w.States.peek().textStyle.format, "I", "")
			w.States.peek().textStyle.format += "I"
		}
	} else {
		switch n.Level {
		case 2:
			w.LogDebug("Strong (leaving)", "")
			w.States.peek().textStyle.format = strings.ReplaceAll(w.States.peek().textStyle.format, "B", "")
		default:
			w.LogDebug("Emph (leaving)", "")
			w.States.peek().textStyle.format = strings.ReplaceAll(w.States.peek().textStyle.format, "I", "")
		}
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderStrikethrough(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.LogDebug("Strike (entering)", "")
		w.States.peek().textStyle.format += "S"
	} else {
		w.LogDebug("Strike (leaving)", "")
		w.States.peek().textStyle.format = strings.ReplaceAll(w.States.peek().textStyle.format, "S", "")
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderTaskCheckBox(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	st := w.GetBacktickStyle()
	if !strings.Contains(strings.ToUpper(st.format), "B") {
		st.format += "B"
	}

	SetStyle(w.Pdf, *st)

	n := node.(*east.TaskCheckBox)
	if n.IsChecked {
		w.Text(*st, "[x] ")
	} else {
		w.Text(*st, "[ ] ")
	}

	return ast.WalkContinue, nil
}

// ensureRowFitsOnPage adds a new page before rendering a table row if the
// row's height would push past the page's bottom margin. Without this,
// gofpdf's auto page break triggers per-cell: cell 0 fits on page N, cell 1
// doesn't, gofpdf adds a page just for cell 1, then the SetY(startY) at the
// end of the cell jumps us back to page N's Y while the active page is N+1.
// Subsequent cells repeat the same pattern, producing one page per cell.
// Pre-breaking the page ensures every cell in the row starts at the same Y
// on the same page.
func ensureRowFitsOnPage(w *Writer, rowHeight float64) {
	_, pageHeight := w.Pdf.GetPageSize()
	_, _, _, bottomMargin := w.Pdf.GetMargins()
	if w.Pdf.GetY()+rowHeight > pageHeight-bottomMargin {
		w.Pdf.AddPage()
	}
}

// tableRowHeight returns the rendered height of the row at curTableRow.
// Header rows are always one line tall; body rows multiply lineHeight by the
// precomputed max-line count. Each row also gets 2*cellPadding of vertical
// breathing room (top + bottom) inside its border. Falls back to a single
// line if curTableRow is past the precomputed list (malformed tables with
// stray rows).
func tableRowHeight(lineHeight float64, isHeader bool) float64 {
	if isHeader || curTableRow >= len(cellMaxLines) {
		return lineHeight + 2*cellPadding
	}
	return lineHeight*float64(cellMaxLines[curTableRow]) + 2*cellPadding
}

func (r *nodeRederFuncs) renderTable(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		currentTableData = CollectTableData(w, source, node)
		cellwidths, cellMaxLines = CalculateTableOptimalColumnWidthsRowHeights(w, currentTableData)
		// Compute the per-cell inset once for the whole table. The width
		// allocator left TBody as the active style, so this uses the body
		// font's m-width; the slight difference from THeader is negligible
		// for visual padding.
		cellPadding = w.Pdf.MeasureTextWidth("m") / 2

		w.LogDebug("Table (entering)", fmt.Sprintf("Column widths: %v, Row line counts: %v", cellwidths, cellMaxLines))

		x := &state{
			containerType: east.KindTable,
			textStyle:     *w.Styles.THeader, listkind: notlist,
			leftMargin: w.States.peek().leftMargin,
		}
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
		w.States.push(x)
		fill = false
	} else {
		wSum := 0.0
		for _, w := range cellwidths {
			wSum += w
		}
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)
		w.Pdf.CellFormat(wSum, 0, "", "T", 0, "", false, 0, "")

		w.States.pop()
		w.LogDebug("Table (leaving)", "")
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)

		currentTableData = nil
		curTableCol = 0
		curTableRow = 0
		cellPadding = 0
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderTableHeader(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.LogDebug("TableHead (entering)", "")
		x := &state{
			containerType: east.KindTableHeader,
			textStyle:     *w.Styles.THeader, listkind: notlist,
			leftMargin: w.States.peek().leftMargin, isHeader: true,
		}
		// If the header row won't fit on the current page, break to a new
		// page now so all of its cells render together. Otherwise gofpdf's
		// auto page break would trigger mid-row, splitting cells across pages.
		ensureRowFitsOnPage(w, x.textStyle.Size+x.textStyle.Spacing+2*cellPadding)
		// Position the cursor at the table's left edge for the header cells.
		// (goldmark puts TableCells directly under TableHeader rather than
		// wrapping them in a TableRow, so the SetX in renderTableRow doesn't
		// fire for the header row.)
		w.Pdf.SetX(x.leftMargin)
		curTableCol = 0
		w.States.push(x)
	} else {
		// Advance Y past the header row before popping. The header isn't a
		// TableRow in goldmark's AST, so renderTableRow's leave-time BR never
		// fires for it; without this, the first body row would render on top
		// of the header.
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing + 2*cellPadding)
		w.States.pop()
		w.LogDebug("TableHead (leaving)", "")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderTableRow(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// `isHeader` is inherited from the parent TableHeader/TableBody state
		// so that header rows pick up the THeader style.
		isHeader := w.States.peek().isHeader
		rowLeftMargin := w.States.peek().leftMargin
		x := &state{
			containerType: east.KindTableRow,
			textStyle:     *w.Styles.TBody, listkind: notlist,
			leftMargin: rowLeftMargin, isHeader: isHeader,
		}
		if isHeader {
			x.textStyle = *w.Styles.THeader
		}
		rowHeight := tableRowHeight(x.textStyle.Size+x.textStyle.Spacing, isHeader)
		w.LogDebug("TableRow (entering)", fmt.Sprintf("isHeader=%v Widths: %v Height: %v", isHeader, cellwidths, rowHeight))

		// Break to a new page before rendering this row if it won't fit on
		// the current one. See ensureRowFitsOnPage for why.
		ensureRowFitsOnPage(w, rowHeight)

		// Explicitly position X at the table's left edge for every row. The
		// row-leave below calls BR(rowHeight) which already returns X to the
		// page's left margin, but the table's left edge can differ from the
		// page margin (e.g., when the table is inside a list or blockquote
		// that pushed SetMarginLeft, or after some other element left the
		// cursor offset). Doing it here makes the row-start invariant
		// independent of prior cursor state.
		w.Pdf.SetX(rowLeftMargin)
		curTableCol = 0
		w.States.push(x)
	} else {
		// All cells in the row have rendered with ln=0, so the cursor is at
		// (rowEndX, rowStartY). Advance Y past the entire row before popping.
		// Without this, the next row would start at startY + lineHeight and
		// overlap multi-line content above it (original bug in issue #31).
		s := w.States.peek()
		w.Pdf.BR(tableRowHeight(s.textStyle.Size+s.textStyle.Spacing, s.isHeader))

		w.States.pop()
		w.LogDebug("TableRow (leaving)", "")
		fill = !fill
		// Only body rows have an entry in cellMaxLines, so only body rows
		// advance the index.
		if !s.isHeader {
			curTableRow++
		}
	}

	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderTableCell(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*east.TableCell)

	if entering {
		w.LogDebug("TableCell (entering)", "")
		isHeader := w.States.peek().isHeader
		x := &state{
			containerType: east.KindTableCell,
			textStyle:     *w.Styles.Normal, listkind: notlist,
			leftMargin: w.States.peek().leftMargin, isHeader: isHeader,
		}
		if isHeader {
			w.Pdf.SetDrawColor(128, 0, 0)
			w.Pdf.SetLineWidth(.3)
			x.isHeader = true
			x.textStyle = *w.Styles.THeader
			SetStyle(w.Pdf, *w.Styles.THeader)
		} else {
			x.textStyle = *w.Styles.TBody
			SetStyle(w.Pdf, *w.Styles.TBody)
			x.isHeader = false
		}
		w.States.push(x)

		text := strings.ReplaceAll(ExtractCellText(source, n), "\n", " ")
		width := cellwidths[curTableCol]
		lineHeight := x.textStyle.Size + x.textStyle.Spacing

		// cellPadding (set in renderTable entering) gives the text visible
		// breathing room from all four cell borders. The horizontal inset
		// is half an m of slack on each side; the vertical inset is the
		// same value applied above and below the text via top/bottom "pad
		// rows" that draw only the L/R border (and fill).

		if isHeader {
			w.LogDebug("... table header cell", fmt.Sprintf("Width=%v, height=%v", width, lineHeight))
			startX := w.Pdf.GetX()
			totalHeight := lineHeight + 2*cellPadding
			// Draw the cell box (border on all sides + fill) without text.
			w.Pdf.CellFormat(width, totalHeight, "", "1", 0, "L", true, 0, "")
			// Draw the text inset horizontally; CellFormat's default vertical
			// alignment ("M"/middle) centers it within totalHeight, giving
			// the padding as top/bottom whitespace.
			w.Pdf.SetX(startX + cellPadding)
			w.Pdf.CellFormat(width-2*cellPadding, totalHeight, text, "", 0, "L", false, 0, "")
			w.Pdf.SetX(startX + width)
		} else {
			// Body cell rendering, see issue #31.
			//
			// We can't pass the full row height to a single CellFormat call:
			// gofpdf.CellFormat doesn't wrap, so overflowing text would draw on
			// one line and visually escape the cell. Instead we split the text
			// into wrapped lines ourselves and draw each on its own row of the
			// cell. cellMaxLines (precomputed in table.go) gives the tallest
			// cell in this row, so all cells in the row render the same number
			// of "line slots" — short cells fill trailing slots with empty
			// text but still draw the L/R border.
			maxLines := 1
			if curTableRow < len(cellMaxLines) {
				maxLines = cellMaxLines[curTableRow]
			}
			w.LogDebug("... table body cell", fmt.Sprintf("Width=%v, lines=%v", width, maxLines))
			// splitWidth must match the wrap width table.go assumed when
			// precomputing cellMaxLines (also width - m), so the row height
			// fits the rendered text.
			splitWidth := width - 2*cellPadding
			if splitWidth < 1 {
				splitWidth = width
			}
			lines := w.Pdf.SplitText(text, splitWidth)

			// The cell renders as: top-pad row, one row per wrapped line, then
			// bottom-pad row. Each chunk draws only the L/R border (and fill),
			// so the visible left/right edges stack into a continuous border;
			// the top/bottom borders come from the surrounding cells (header
			// box above, table's closing T-border below). For each text row we
			// do two CellFormat calls — a border+fill pass at full width, then
			// a text-only pass inset by `cellPadding` — to give the text
			// visible left/right padding inside the L/R borders.
			startX := w.Pdf.GetX()
			startY := w.Pdf.GetY()

			// Top padding row. SetY must come before SetX because SetY resets
			// X to the page's left margin in gofpdf, which would otherwise
			// undo SetX(startX) and make every non-first cell's loop draw
			// at the wrong X (overpainting the previous cell with the row's
			// fill color).
			w.Pdf.CellFormat(width, cellPadding, "", "LR", 0, "", fill, 0, "")
			w.Pdf.SetY(startY + cellPadding)
			w.Pdf.SetX(startX)

			// Per-line border+text passes.
			for i := 0; i < maxLines; i++ {
				if i > 0 {
					w.Pdf.BR(lineHeight)
					w.Pdf.SetX(startX)
				}
				lineText := ""
				if i < len(lines) {
					lineText = lines[i]
				}
				w.Pdf.CellFormat(width, lineHeight, "", "LR", 0, "", fill, 0, "")
				w.Pdf.SetX(startX + cellPadding)
				w.Pdf.CellFormat(width-2*cellPadding, lineHeight, lineText, "", 0, "", false, 0, "")
			}

			// Bottom padding row.
			w.Pdf.BR(lineHeight)
			w.Pdf.SetX(startX)
			w.Pdf.CellFormat(width, cellPadding, "", "LR", 0, "", fill, 0, "")

			// Restore cursor to the row's top so the next sibling cell starts
			// at the same baseline.
			w.Pdf.SetY(startY)
			w.Pdf.SetX(startX + width)
		}
	} else {
		w.States.pop()
		w.LogDebug("TableCell (leaving)", "")
		curTableCol++
	}

	return ast.WalkSkipChildren, nil
}
