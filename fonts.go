package pdf

import (
	"context"
	"fmt"

	"github.com/go-swiss/fonts"
	"github.com/go-swiss/fonts/google"
)

type fontType string

const (
	FontTypeInbuilt fontType = "inbuilt_font"
	FontTypeCustom  fontType = "custom_font"
	FontTypeGoogle  fontType = "google_font"
)

// A map of the inbuilt fonts that should be used for text (Headings, body)
var TextFontsInbuilt = map[string]Font{FontTimes.Family: FontTimes, FontHelvetica.Family: FontHelvetica}

// A map of the inbuilt monospace fonts. To be used for code blocks
var CodeFontsInbuilt = map[string]Font{FontCourier.Family: FontCourier}

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

// Returns a font from one of TextFontsInbuilt and TextFontsGoogle, or the backup
func GetTextFont(family string, fallback Font) Font {
	f, ok := TextFontsInbuilt[family]
	if ok {
		return f
	}

	f, ok = TextFontsGoogle[family]
	if ok {
		return f
	}

	return fallback
}

// Returns a font from CodeFontsInbuilt and CodeFontsGoogle, or the backup
func GetCodeFont(family string, fallback Font) Font {
	f, ok := CodeFontsInbuilt[family]
	if ok {
		return f
	}

	f, ok = CodeFontsGoogle[family]
	if ok {
		return f
	}

	return fallback
}

func addStyleFonts(ctx context.Context, pdf PDF, styles Styles, fontsCache fonts.Cache) error {
	fontsToLoad := []Font{
		styles.H1.Font,
		styles.H2.Font,
		styles.H3.Font,
		styles.H4.Font,
		styles.H5.Font,
		styles.H6.Font,

		styles.Normal.Font,
		styles.Blockquote.Font,

		styles.THeader.Font,
		styles.TBody.Font,

		styles.CodeFont,
	}

	return AddFonts(ctx, pdf, fontsToLoad, fontsCache)
}

func AddFonts(ctx context.Context, pdf PDF, fonts []Font, fontsCache fonts.Cache) error {
	// Create a cache
	if fontsCache == nil {
		fontsCache = defaultCache
	}

	getFontBytes := func(family, variant string) ([]byte, error) {
		fontBytes, err := google.GetFontBytes(ctx, family, variant, fontsCache)
		if err != nil {
			return nil, fmt.Errorf("could not get font bytes. %s-%s: %w", family, variant, err)
		}

		return fontBytes, nil
	}

	for _, f := range fonts {
		if f.Type == FontTypeInbuilt || f.Type == FontTypeCustom {
			continue
		}

		regular, err := getFontBytes(f.Family, f.FileRegular)
		if err != nil {
			return err
		}
		italic, err := getFontBytes(f.Family, f.FileItalic)
		if err != nil {
			return err
		}
		bold, err := getFontBytes(f.Family, f.FileBold)
		if err != nil {
			return err
		}
		boldItalic, err := getFontBytes(f.Family, f.FileBoldItalic)
		if err != nil {
			return err
		}

		if err := pdf.AddFont(f.Family, FontStyleRegular, regular); err != nil {
			return err
		}
		if err := pdf.AddFont(f.Family, FontStyleItalic, italic); err != nil {
			return err
		}
		if err := pdf.AddFont(f.Family, FontStyleBold, bold); err != nil {
			return err
		}
		if err := pdf.AddFont(f.Family, FontStyleBoldItalic, boldItalic); err != nil {
			return err
		}
	}

	return nil
}

const (
	FontStyleRegular    = ""
	FontStyleBold       = "B"
	FontStyleItalic     = "I"
	FontStyleBoldItalic = "BI"
)
