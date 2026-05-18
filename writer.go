package pdf

import (
	"bufio"
	"bytes"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/util"
)

// Holds the neccessary information to write to a PDF
type Writer struct {
	Pdf     PDF
	ImageFS http.FileSystem
	Styles  Styles
	States  states

	EscapeHTML  bool
	DebugWriter io.Writer
}

// To log debug information. Nothing is logged if DebugWriter is nil
func (r *Writer) LogDebug(source, msg string) {
	if r.DebugWriter != nil {
		indent := strings.Repeat("-", len(r.States.stack)-1)
		_, _ = fmt.Fprintf(r.DebugWriter, "%v[%v] %v\n", indent, source, msg)
	}
}

// Write a string with a given style
func (w *Writer) Text(s Style, t string) {
	newString := ""
	for _, r := range string(t) {
		if int(r) < 65535 {
			newString += string(r)
		}
	}

	w.Pdf.WriteText(s.Size+s.Spacing, newString)
}

// Write a link
func (w *Writer) WriteLink(s Style, display, url string) {
	if url[0] == '#' {
		w.Pdf.WriteInternalLink(s.Size+s.Spacing, display, url[1:])
		return
	}
	w.Pdf.WriteExternalLink(s.Size+s.Spacing, display, url)
}

// WriteText based on the current state.
func (w *Writer) WriteText(stringContents string) {
	bb := &bytes.Buffer{}
	bufw := bufio.NewWriter(bb)

	stringContents = strings.ReplaceAll(stringContents, "\n", " ")
	w.LogDebug("Text", stringContents)

	currentStyle := w.States.peek().textStyle
	SetStyle(w.Pdf, currentStyle)

	if w.States.peek().destination != "" {
		w.WriteLink(currentStyle, stringContents, w.States.peek().destination)
	} else if w.States.peek().containerType == ast.KindHeading {
		escapeWriter{escapeHTML: w.EscapeHTML}.write(bufw, []byte(stringContents))
	} else if w.States.peek().containerType == east.KindTableCell {
		width := cellwidths[curTableCol]
		lineHeight := currentStyle.Size + currentStyle.Spacing

		if w.States.peek().isHeader {
			w.LogDebug("... table header cell", fmt.Sprintf("Width=%v, height=%v", width, lineHeight))
			w.Pdf.CellFormat(width, lineHeight, stringContents, "1", 0, "L", true, 0, "")
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

			// The column allocator (table.go) sized this column as text + 2*m
			// of slack. Use half of that as wrap padding so the rendered text
			// stops short of the right border without consuming all the slack —
			// the remaining 1m keeps long words from being chopped mid-character
			// when columns get squeezed by the available-width fit.
			splitWidth := width - w.Pdf.MeasureTextWidth("m")
			if splitWidth < 1 {
				splitWidth = width
			}
			lines := w.Pdf.SplitText(stringContents, splitWidth)

			// Each CellFormat below advances X by `width` but leaves Y alone
			// (ln=0). We use BR+SetX to step down within this cell, then on
			// exit restore Y to the row's top and bump X past the cell so the
			// next sibling cell renders at the same baseline. Without this
			// reset, taller cells would push neighbors down and break the row.
			startX := w.Pdf.GetX()
			startY := w.Pdf.GetY()
			for i := 0; i < maxLines; i++ {
				if i > 0 {
					w.Pdf.BR(lineHeight)
					w.Pdf.SetX(startX)
				}
				lineText := ""
				if i < len(lines) {
					lineText = lines[i]
				}
				w.Pdf.CellFormat(width, lineHeight, lineText, "LR", 0, "", fill, 0, "")
			}
			w.Pdf.SetY(startY)
			w.Pdf.SetX(startX + width)
		}
	} else {
		escapeWriter{escapeHTML: w.EscapeHTML}.write(bufw, []byte(stringContents))
	}

	if bufw.Size() > 0 {
		_ = bufw.Flush()
		w.Text(currentStyle, bb.String())
	}
}

// Converts a chroma.StyleEntry to a Style, based on the backtickStyle
func (w *Writer) ChromaToStyle(chSt chroma.StyleEntry) *Style {
	s := w.GetBacktickStyle()
	s.TextColor = color.RGBA{
		R: chSt.Colour.Red(),
		G: chSt.Colour.Green(),
		B: chSt.Colour.Blue(),
	}
	s.FillColor = color.RGBA{
		R: chSt.Background.Red(),
		G: chSt.Background.Green(),
		B: chSt.Background.Blue(),
	}

	styleStr := strings.ToUpper(s.format)
	if chSt.Bold == chroma.Yes && !strings.Contains(styleStr, "B") {
		s.format += "B"
	}

	if chSt.Italic == chroma.Yes && !strings.Contains(styleStr, "I") {
		s.format += "I"
	}

	if chSt.Underline == chroma.Yes && !strings.Contains(styleStr, "U") {
		s.format += "U"
	}

	return s
}

// Returns the current style with the font set to Styles.CodeFont
func (w *Writer) GetBacktickStyle() *Style {
	st := w.States.peek().textStyle
	st.Font = w.Styles.CodeFont

	return &st
}

// Returns the currentStyle adding underline and setting the color to Styles.LinkColor
func (w *Writer) GetLinkStyle() *Style {
	st := w.States.peek().textStyle

	if w.Styles.LinkColor != nil {
		st.TextColor = w.Styles.LinkColor
	}

	styleStr := strings.ToUpper(st.format)
	if !strings.Contains(styleStr, "U") {
		st.format += "U"
	}

	return &st
}

type escapeWriter struct {
	escapeHTML bool
}

func (e escapeWriter) escapeRune(writer util.BufWriter, r rune) {
	if r < 256 {
		v := util.EscapeHTMLByte(byte(r))
		if v != nil && e.escapeHTML {
			_, _ = writer.Write(v)
			return
		}
	}
	_, _ = writer.WriteRune(util.ToValidRune(r))
}

func (e escapeWriter) write(writer util.BufWriter, source []byte) {
	escaped := false
	var ok bool
	limit := len(source)
	n := 0
	for i := 0; i < limit; i++ {
		c := source[i]
		if escaped {
			if util.IsPunct(c) {
				e.rawWrite(writer, source[n:i-1])
				n = i
				escaped = false
				continue
			}
		}
		if c == '&' {
			pos := i
			next := i + 1
			if next < limit && source[next] == '#' {
				nnext := next + 1
				if nnext < limit {
					nc := source[nnext]
					// code point like #x22;
					if nnext < limit && nc == 'x' || nc == 'X' {
						start := nnext + 1
						i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsHexDecimal)
						if ok && i < limit && source[i] == ';' {
							v, _ := strconv.ParseUint(util.BytesToReadOnlyString(source[start:i]), 16, 32)
							e.rawWrite(writer, source[n:pos])
							n = i + 1
							e.escapeRune(writer, rune(v))
							continue
						}
						// code point like #1234;
					} else if nc >= '0' && nc <= '9' {
						start := nnext
						i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsNumeric)
						if ok && i < limit && i-start < 8 && source[i] == ';' {
							v, _ := strconv.ParseUint(util.BytesToReadOnlyString(source[start:i]), 0, 32)
							e.rawWrite(writer, source[n:pos])
							n = i + 1
							e.escapeRune(writer, rune(v))
							continue
						}
					}
				}
			} else {
				start := next
				i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsAlphaNumeric)
				// entity reference
				if ok && i < limit && source[i] == ';' {
					name := util.BytesToReadOnlyString(source[start:i])
					entity, ok := util.LookUpHTML5EntityByName(name)
					if ok {
						e.rawWrite(writer, source[n:pos])
						n = i + 1
						e.rawWrite(writer, entity.Characters)
						continue
					}
				}
			}
			i = next - 1
		}
		if c == '\\' {
			escaped = true
			continue
		}
		escaped = false
	}
	e.rawWrite(writer, source[n:])
}

func (e escapeWriter) rawWrite(writer util.BufWriter, source []byte) {
	n := 0
	l := len(source)
	for i := 0; i < l; i++ {
		v := util.EscapeHTMLByte(source[i])
		if v != nil && e.escapeHTML {
			_, _ = writer.Write(source[i-n : i])
			n = 0
			_, _ = writer.Write(v)
			continue
		}
		n++
	}
	if n != 0 {
		_, _ = writer.Write(source[l-n:])
	}
}
