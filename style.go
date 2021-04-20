package pdf

import (
	"image/color"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/styles"
)

// Style is the struct to capture the styling features for text
// Size and Spacing are specified in points.
// The sum of Size and Spacing is used as line height value
// in the gofpdf API
type Style struct {
	Font      Font
	Size      float64
	Spacing   float64
	TextColor color.Color
	FillColor color.Color

	// For formatting the text
	format string
}

type Styles struct {
	// Headings
	H1 *Style
	H2 *Style
	H3 *Style
	H4 *Style
	H5 *Style
	H6 *Style

	// normal text
	Normal *Style

	// blockquote text
	Blockquote *Style

	// Table styling
	THeader *Style
	TBody   *Style

	// code and preformatted text
	CodeFont Font
	// Codeblock Chroma Theme
	CodeBlockTheme *chroma.Style

	// link text
	LinkColor color.Color

	// IndentValue float64
}

func DefaultStyles() Styles {
	c := Styles{}

	c.Normal = &Style{Font: FontRoboto, Size: 12, Spacing: 3,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0}}

	c.H1 = &Style{Font: FontRoboto, Size: 24, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0}}
	c.H2 = &Style{Font: FontRoboto, Size: 22, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0}}
	c.H3 = &Style{Font: FontRoboto, Size: 20, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0}}
	c.H4 = &Style{Font: FontRoboto, Size: 18, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0}}
	c.H5 = &Style{Font: FontRoboto, Size: 16, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0}}
	c.H6 = &Style{Font: FontRoboto, Size: 14, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0}}

	c.Blockquote = &Style{Font: FontRoboto, Size: 14, Spacing: 1,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0}}

	c.THeader = &Style{Font: FontRoboto, Size: 12, Spacing: 2,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{180, 180, 180, 0}}
	c.TBody = &Style{Font: FontRoboto, Size: 12, Spacing: 2,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{240, 240, 240, 0}}

	c.CodeFont = FontRobotoMono
	c.CodeBlockTheme = styles.Register(chroma.MustNewStyle("custom", chroma.StyleEntries{
		chroma.Background:           "#cccccc bg:#1d1d1d",
		chroma.Comment:              "bold #999999",
		chroma.CommentSpecial:       "bold #cd0000",
		chroma.Keyword:              "#cc99cd",
		chroma.KeywordDeclaration:   "#cc99cd",
		chroma.KeywordNamespace:     "#cc99cd",
		chroma.KeywordType:          "#cc99cd",
		chroma.Operator:             "#67cdcc",
		chroma.OperatorWord:         "#cdcd00",
		chroma.NameClass:            "#f08d49",
		chroma.NameBuiltin:          "#f08d49",
		chroma.NameFunction:         "#f08d49",
		chroma.NameException:        "bold #666699",
		chroma.NameVariable:         "#00cdcd",
		chroma.LiteralString:        "#7ec699",
		chroma.LiteralNumber:        "#f08d49",
		chroma.LiteralStringBoolean: "#f08d49",
		chroma.GenericHeading:       "bold #000080",
		chroma.GenericSubheading:    "bold #800080",
		chroma.GenericDeleted:       "#e2777a",
		chroma.GenericInserted:      "#cc99cd",
		chroma.GenericError:         "#e2777a",
		chroma.GenericEmph:          "italic",
		chroma.GenericStrong:        "bold",
		chroma.GenericPrompt:        "bold #000080",
		chroma.GenericOutput:        "#888",
		chroma.GenericTraceback:     "#04D",
		chroma.GenericUnderline:     "underline",
		chroma.Error:                "border:#e2777a",
	}))

	return c
}
