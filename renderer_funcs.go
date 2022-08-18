package pdf

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
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

		switch n.Level {
		case 1:
			w.LogDebug("Heading (1, entering)", "")
			x := &state{
				containerType: n.Kind(),
				textStyle:     *w.Styles.H1, listkind: notlist,
				leftMargin: w.States.peek().leftMargin,
			}
			w.States.push(x)
		case 2:
			w.LogDebug("Heading (2, entering)", "")
			x := &state{
				containerType: n.Kind(),
				textStyle:     *w.Styles.H2, listkind: notlist,
				leftMargin: w.States.peek().leftMargin,
			}
			w.States.push(x)
		case 3:
			w.LogDebug("Heading (3, entering)", "")
			x := &state{
				containerType: n.Kind(),
				textStyle:     *w.Styles.H3, listkind: notlist,
				leftMargin: w.States.peek().leftMargin,
			}
			w.States.push(x)
		case 4:
			w.LogDebug("Heading (4, entering)", "")
			x := &state{
				containerType: n.Kind(),
				textStyle:     *w.Styles.H4, listkind: notlist,
				leftMargin: w.States.peek().leftMargin,
			}
			w.States.push(x)
		case 5:
			w.LogDebug("Heading (5, entering)", "")
			x := &state{
				containerType: n.Kind(),
				textStyle:     *w.Styles.H5, listkind: notlist,
				leftMargin: w.States.peek().leftMargin,
			}
			w.States.push(x)
		case 6:
			w.LogDebug("Heading (6, entering)", "")
			x := &state{
				containerType: n.Kind(),
				textStyle:     *w.Styles.H6, listkind: notlist,
				leftMargin: w.States.peek().leftMargin,
			}
			w.States.push(x)
		}
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
		if _, ok := node.NextSibling().(ast.Node); ok && node.FirstChild() != nil {
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

				if k != len(tokenLines)-1 || v == "" {
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

	// do a newline
	w.Pdf.BR(LH)
	// get the current x and y (assume left margin in ok)
	x := w.Pdf.GetX()
	y := w.Pdf.GetY()
	// get the page margins
	_, _, rm, _ := w.Pdf.GetMargins()
	// get the page size
	width, _ := w.Pdf.GetPageSize()
	// now compute the x value of the right side of page
	newx := width - rm

	w.Pdf.SetLineWidth(3)
	w.Pdf.SetFillColor(200, 200, 200)

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
			textStyle:     *(w.GetLinkStyle()), listkind: notlist,
			leftMargin:  w.States.peek().leftMargin,
			destination: string(n.Destination),
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
		textStyle:     *(w.GetLinkStyle()), listkind: notlist,
		leftMargin:  w.States.peek().leftMargin,
		destination: destination,
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
			textStyle:     *(w.GetBacktickStyle()), listkind: notlist,
			leftMargin: w.States.peek().leftMargin, destination: w.States.peek().destination,
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
		imgPath := string(n.Destination)
		imgFile, err := w.ImageFS.Open(imgPath)
		if err == nil {
			defer imgFile.Close()

			width, _ := w.Pdf.GetPageSize()
			mleft, _, mright, _ := w.Pdf.GetMargins()
			maxw := width - (mleft * 2) - (mright * 2)

			format := strings.ToUpper(strings.Trim(filepath.Ext(imgPath), "."))
			w.Pdf.RegisterImage(imgPath, format, imgFile)
			w.Pdf.UseImage(imgPath, (mleft * 2), w.Pdf.GetY(), maxw, 0)
		} else {
			log.Printf("IMAGE ERROR: %v", err)
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
			w.States.peek().textStyle.format = strings.Replace(w.States.peek().textStyle.format, "B", "", -1)
			w.States.peek().textStyle.format += "B"
		default:
			w.LogDebug("Emph (entering)", "")
			w.States.peek().textStyle.format = strings.Replace(w.States.peek().textStyle.format, "I", "", -1)
			w.States.peek().textStyle.format += "I"
		}
	} else {
		switch n.Level {
		case 2:
			w.LogDebug("Strong (leaving)", "")
			w.States.peek().textStyle.format = strings.Replace(w.States.peek().textStyle.format, "B", "", -1)
		default:
			w.LogDebug("Emph (leaving)", "")
			w.States.peek().textStyle.format = strings.Replace(w.States.peek().textStyle.format, "I", "", -1)
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
		w.States.peek().textStyle.format = strings.Replace(w.States.peek().textStyle.format, "S", "", -1)
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

func (r *nodeRederFuncs) renderTable(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.LogDebug("Table (entering)", "")
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
		w.States.push(x)
		cellwidths = make([]float64, 0)
	} else {
		w.States.pop()
		w.LogDebug("TableHead (leaving)", "")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRederFuncs) renderTableRow(w *Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.LogDebug("TableRow (entering)", "")
		isHeader := w.States.peek().isHeader
		x := &state{
			containerType: east.KindTableRow,
			textStyle:     *w.Styles.TBody, listkind: notlist,
			leftMargin: w.States.peek().leftMargin, isHeader: isHeader,
		}
		if w.States.peek().isHeader {
			x.textStyle = *w.Styles.THeader
		}
		w.Pdf.BR(w.States.peek().textStyle.Size + w.States.peek().textStyle.Spacing)

		// initialize cell widths slice; only one table at a time!
		curdatacell = 0
		w.States.push(x)
	} else {
		w.States.pop()
		w.LogDebug("TableRow (leaving)", "")
		fill = !fill
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

		w.WriteText(string(n.Text(source)))
	} else {
		w.States.pop()
		w.LogDebug("TableCell (leaving)", "")
		curdatacell++
	}

	return ast.WalkSkipChildren, nil
}
