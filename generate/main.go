package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dave/jennifer/jen"
	"google.golang.org/api/option"
	"google.golang.org/api/webfonts/v1"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	apikey := os.Getenv("GOOGLE_FONTS_API_KEY")
	if apikey == "" {
		log.Fatalf("env var GOOGLE_FONTS_API_KEY is needed to generate files")
	}

	ws, err := webfonts.NewService(ctx, option.WithAPIKey(apikey))
	if err != nil {
		log.Fatalf("could not get webfonts service: %v", err)
	}

	fonts, err := webfonts.NewWebfontsService(ws).List().Context(ctx).Do()
	if err != nil {
		log.Fatalf("could not get webfonts: %v", err)
	}

	err = generateConstants(fonts)
	if err != nil {
		log.Fatalf("could not generate constants: %v", err)
	}
}

func generateConstants(fonts *webfonts.WebfontList) error {
	// Generate our fonts.go file
	f := jen.NewFilePathName("github.com/stephenafamo/goldmark-pdf", "pdf")

	f.HeaderComment("Code generated, DO NOT EDIT.")
	// f.HeaderComment("// +build google")
	f.ImportName("github.com/stephenafamo/fonts", "fonts")

	gTextFonts := f.Var().Id("TextFontsGoogle").Op("=").Map(jen.String()).Qual("github.com/stephenafamo/goldmark-pdf", "Font")
	f.Line()
	gCodeFonts := f.Var().Id("CodeFontsGoogle").Op("=").Map(jen.String()).Qual("github.com/stephenafamo/goldmark-pdf", "Font")
	f.Line()

	bodyFonts := make(jen.Dict)
	codeFonts := make(jen.Dict)
	// create variables for the fonts we use
	for _, font := range fonts.Items {
		var hasRegular, hasItalic, hasBold, hasBoldItalic bool
		regular, italic, bold, boldItalic := "regular", "italic", "700", "700italic"

		for _, variant := range font.Variants {
			switch variant {
			case regular:
				hasRegular = true
			case italic:
				hasItalic = true
			case bold:
				hasBold = true
			case boldItalic:
				hasBoldItalic = true
			}
		}

		canUseForText := hasRegular && hasItalic && hasBold && hasBoldItalic
		canUseForCode := font.Category == "monospace"

		if !canUseForText && !canUseForCode {
			continue
		}

		// Populate any missing files...
		// useful for monospace fonts that do not have all weights
		if !hasItalic {
			italic = regular
		}

		if !hasBold {
			bold = regular
		}

		if !hasBoldItalic {
			boldItalic = bold
		}

		family := strings.ReplaceAll(font.Family, " ", "")
		variableName := "Font" + family

		f.Var().Id(variableName).Op("=").
			Qual("github.com/stephenafamo/goldmark-pdf", "Font").Values(jen.Dict{
			jen.Id("CanUseForText"): jen.Lit(canUseForText),
			jen.Id("CanUseForCode"): jen.Lit(canUseForCode),

			jen.Id("Category"): jen.Lit(font.Category),
			jen.Id("Family"):   jen.Lit(font.Family),

			jen.Id("FileRegular"):    jen.Lit(regular),
			jen.Id("FileItalic"):     jen.Lit(italic),
			jen.Id("FileBold"):       jen.Lit(bold),
			jen.Id("FileBoldItalic"): jen.Lit(boldItalic),

			jen.Id("Type"): jen.Qual("github.com/stephenafamo/goldmark-pdf", "FontTypeGoogle"),
		})
		f.Line()

		if canUseForText {
			key := jen.Qual("github.com/stephenafamo/goldmark-pdf", variableName+".Family")
			val := jen.Qual("github.com/stephenafamo/goldmark-pdf", variableName)
			bodyFonts[key] = val
		}

		if canUseForCode {
			key := jen.Qual("github.com/stephenafamo/goldmark-pdf", variableName+".Family")
			val := jen.Qual("github.com/stephenafamo/goldmark-pdf", variableName)
			codeFonts[key] = val
		}
	}

	gTextFonts.Values(bodyFonts)
	gCodeFonts.Values(codeFonts)

	return f.Save("fonts_google.go")
}
