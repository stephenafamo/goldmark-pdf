# goldmark-pdf

goldmark-pdf is a renderer for [goldmark](http://github.com/yuin/goldmark) that allows rendering to PDF.

![goldmark-pdf screenshot](https://res.cloudinary.com/stephenafamo/image/upload/v1618945448/goldmark-pdf%20screenshot.png)

## Reference

See <https://pkg.go.dev/github.com/stephenafamo/goldmark-pdf>

## Usage

Care has been taken to match the semantics of goldmark and its extensions.

The PDF renderer can be initiated with `pdf.New()` and the returned value satisfies `goldmark`'s `renderer.Renderer` interface, so it can be passed to `goldmark.New()` using the `goldmark.WithRenderer()` option.

```go
markdown := goldmark.New(
    goldmark.WithRenderer(pdf.New()),
)
```

Options can also be passed to `pdf.New()`, the options interface to be satisfied is:

```go
// An Option interface is a functional option type for the Renderer.
type Option interface {
	SetConfig(*Config)
}
```

Here is the `Config` struct that is to be modified:

```go
type Config struct {
	Context context.Context

	PDF PDF

	// A source for images
	ImageFS fs.FS

	// All other options have sensible defaults
	Styles Styles

	// A cache for the fonts
	FontsCache fonts.Cache

	// For debugging
	TraceWriter io.Writer

	NodeRenderers util.PrioritizedSlice
}
```

> Some helper functions for adding options are already provided. See [`option.go`](https://github.com/stephenafamo/goldmark-pdf/blob/master/option.go)

An example with some more options: 

```go
goldmark.New(
    goldmark.WithRenderer(
        pdf.New(
            pdf.WithTraceWriter(os.Stdout),
            pdf.WithContext(context.Background()),
            pdf.WithImageFS(os.DirFS(".")),
            pdf.WithLinkColor("cc4578"),
            pdf.WithHeadingFont(pdf.GetTextFont("IBM Plex Serif", pdf.FontLora)),
            pdf.WithBodyFont(pdf.GetTextFont("Open Sans", pdf.FontRoboto)),
            pdf.WithCodeFont(pdf.GetCodeFont("Inconsolata", pdf.FontRobotoMono)),
            pdf.WithEscapeHTML(false), // default: true
        ),
    ),
    goldmark.WithRendererOptions(
        html.WithUnsafe(), // for compatibility if WithEscapeHTML() is not used
    ),
)
```

## Fonts

The fonts that can be used in the PDF are based on the `Font` struct

```go
// Represents a font.
type Font struct {
	CanUseForText bool
	CanUseForCode bool

	Category string
	Family   string

	FileRegular    string
	FileItalic     string
	FileBold       string
	FileBoldItalic string

	Type fontType
}
```

To be used for text, a font should have regular, italic, bold and bold-italic styles. Each of these has to be loaded separately.

To ease this process, variables have been generated for all the Google fonts that have these styles. For example: 

```go
var FontRoboto = Font{
	CanUseForCode:  false,
	CanUseForText:  true,
	Category:       "sans-serif",
	Family:         "Roboto",
	FileBold:       "700",
	FileBoldItalic: "700italic",
	FileItalic:     "italic",
	FileRegular:    "regular",
	Type:           fontTypeGoogle,
}
```

For codeblocks, if any other style is missing, the regular font is used in place.

```go
var FontMajorMonoDisplay = Font{
	CanUseForCode:  true,
	CanUseForText:  false,
	Category:       "monospace",
	Family:         "Major Mono Display",
	FileBold:       "regular",
	FileBoldItalic: "regular",
	FileItalic:     "regular",
	FileRegular:    "regular",
	Type:           fontTypeGoogle,
}
```

When loading the fonts, they are downloaded on the fly using the [`fonts`](https://github.com/go-swiss/fonts).

If you'd like to use a font outside of these, you should pass your own font struct which have been loaded into the `PDF` object you set in the `Config`. Be sure to set the `FontType` to `FontTypeCustom` so that we do not attempt to download it.

## Contributing

Here's a list of things that I'd love help with:

* [ ] More documentation
* [ ] Testing
* [ ] Finish the (currently buggy) implementation based on [`gopdf`](https://github.com/signintech/gopdf)


## License

MIT

## Author 

[Stephen Afam-Osemene](https://stephenafamo.com)
