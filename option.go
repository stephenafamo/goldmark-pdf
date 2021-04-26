package pdf

import (
	"context"
	"image/color"
	"io"
	"io/fs"

	"github.com/alecthomas/chroma"
	"github.com/go-swiss/fonts"
	"github.com/yuin/goldmark/util"
)

// An Option interface is a functional option type for the Renderer.
type Option interface {
	SetConfig(*Config)
}

// A function that implements the Option interface
type OptionFunc func(*Config)

// To implement the SetConfig method of the Option interface
func (o OptionFunc) SetConfig(c *Config) {
	o(c)
}

// Pass a completely new config
func WithConfig(config *Config) Option {
	return OptionFunc(func(c *Config) {
		*c = *config
	})
}

// Add a context that will be used for operations like downloading fonts
func WithContext(ctx context.Context) Option {
	return OptionFunc(func(c *Config) {
		c.Context = ctx
	})
}

// Set the image filesystem
func WithImageFS(images fs.FS) Option {
	return OptionFunc(func(c *Config) {
		c.ImageFS = images
	})
}

// Set a cache for fonts
func WithFontsCache(fc fonts.Cache) Option {
	return OptionFunc(func(c *Config) {
		c.FontsCache = fc
	})
}

// Set a color for links
func WithLinkColor(val color.Color) Option {
	return OptionFunc(func(c *Config) {
		c.Styles.LinkColor = val
	})
}

// Provide an io.Write where debug information will be written to
func WithTraceWriter(val io.Writer) Option {
	return OptionFunc(func(c *Config) {
		c.TraceWriter = val
	})
}

// Set the font for every heading style
func WithHeadingFont(f Font) Option {
	return OptionFunc(func(c *Config) {
		c.Styles.H1.Font = f
		c.Styles.H2.Font = f
		c.Styles.H3.Font = f
		c.Styles.H4.Font = f
		c.Styles.H5.Font = f
		c.Styles.H6.Font = f
		c.Styles.THeader.Font = f
	})
}

// Set the font for every body element
func WithBodyFont(f Font) Option {
	return OptionFunc(func(c *Config) {
		c.Styles.Normal.Font = f
		c.Styles.Blockquote.Font = f
		c.Styles.TBody.Font = f
	})
}

// Set a font for code spans and code blocks
func WithCodeFont(f Font) Option {
	return OptionFunc(func(c *Config) {
		c.Styles.CodeFont = f
	})
}

// Set the code block chroma theme
func WithCodeBlockTheme(theme *chroma.Style) Option {
	return OptionFunc(func(c *Config) {
		c.Styles.CodeBlockTheme = theme
	})
}

type withNodeRenderers struct {
	value []util.PrioritizedValue
}

func (o *withNodeRenderers) SetConfig(c *Config) {
	c.NodeRenderers = append(c.NodeRenderers, o.value...)
}

// Extend the NodeRenderers to support or overwrite how nodes are rendered.
func WithNodeRenderers(ps ...util.PrioritizedValue) Option {
	return &withNodeRenderers{ps}
}

// Pass your own PDF object that satisfies the PDF interface
func WithPDF(pdf PDF) Option {
	return OptionFunc(func(c *Config) {
		c.PDF = pdf
	})
}

// Easily configure a PDF writer to use based on https://github.com/phpdave11/gofpdf
func WithFpdf(ctx context.Context, c FpdfConfig) Option {
	return WithPDF(NewFpdf(ctx, c, nil))
}
