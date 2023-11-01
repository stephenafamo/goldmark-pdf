package pdf

import (
	"image/color"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
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

	c.Normal = &Style{
		Font: FontRoboto, Size: 12, Spacing: 3,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0},
	}

	c.H1 = &Style{
		Font: FontRoboto, Size: 24, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0},
	}
	c.H2 = &Style{
		Font: FontRoboto, Size: 22, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0},
	}
	c.H3 = &Style{
		Font: FontRoboto, Size: 20, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0},
	}
	c.H4 = &Style{
		Font: FontRoboto, Size: 18, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0},
	}
	c.H5 = &Style{
		Font: FontRoboto, Size: 16, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0},
	}
	c.H6 = &Style{
		Font: FontRoboto, Size: 14, Spacing: 5,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0},
	}

	c.Blockquote = &Style{
		Font: FontRoboto, Size: 14, Spacing: 1,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{255, 255, 255, 0},
	}

	c.THeader = &Style{
		Font: FontRoboto, Size: 12, Spacing: 2,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{180, 180, 180, 0},
	}
	c.TBody = &Style{
		Font: FontRoboto, Size: 12, Spacing: 2,
		TextColor: color.RGBA{0, 0, 0, 0}, FillColor: color.RGBA{240, 240, 240, 0},
	}

	c.CodeFont = FontRobotoMono
	c.CodeBlockTheme = styles.Get("monokai")

	return c
}
