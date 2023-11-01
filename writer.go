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

	DebugWriter io.Writer
}

// To log debug information. Nothing is logged if DebugWriter is nil
func (r *Writer) LogDebug(source, msg string) {
	if r.DebugWriter != nil {
		indent := strings.Repeat("-", len(r.States.stack)-1)
		r.DebugWriter.Write([]byte(fmt.Sprintf("%v[%v] %v\n", indent, source, msg)))
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

// Write text based on the current state.
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
		escapeWriter{}.write(bufw, []byte(stringContents))
	} else if w.States.peek().containerType == east.KindTableCell {
		if w.States.peek().isHeader {
			SetStyle(w.Pdf, currentStyle)
			// get the string width of header value
			hw := w.Pdf.MeasureTextWidth(stringContents) + (2 * w.Pdf.MeasureTextWidth("m"))
			// now append it
			cellwidths = append(cellwidths, hw)
			// now write it...
			h := currentStyle.Size + currentStyle.Spacing
			w.LogDebug("... table header cell", fmt.Sprintf("Width=%v, height=%v", hw, h))

			w.Pdf.CellFormat(hw, h, stringContents, "1", 0, "L", true, 0, "")
		} else {
			SetStyle(w.Pdf, currentStyle)
			hw := cellwidths[curdatacell]
			h := currentStyle.Size + currentStyle.Spacing
			w.LogDebug("... table body cell", fmt.Sprintf("Width=%v, height=%v", hw, h))
			w.Pdf.CellFormat(hw, h, stringContents, "LR", 0, "", fill, 0, "")
		}
	} else {
		escapeWriter{}.write(bufw, []byte(stringContents))
	}

	bufw.Flush()
	w.Text(currentStyle, bb.String())
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

type escapeWriter struct{}

func (e escapeWriter) escapeRune(writer util.BufWriter, r rune) {
	if r < 256 {
		v := util.EscapeHTMLByte(byte(r))
		if v != nil {
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
		if v != nil {
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
