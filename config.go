package pdf

import (
	"context"
	goldrender "github.com/yuin/goldmark/renderer"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-swiss/fonts"
	"github.com/yuin/goldmark/util"
)

type Config struct {
	goldrender.Config
	Context context.Context

	PDF PDF

	// A source for images
	ImageFS http.FileSystem

	// All other options have sensible defaults
	Styles Styles

	// A cache for the fonts
	FontsCache fonts.Cache

	// Logger receives trace and warning events from the renderer. A nil
	// Logger disables all logging — use WithLogger(slog.Default()) (or any
	// configured logger) to enable.
	Logger *slog.Logger

	NodeRenderers util.PrioritizedSlice
}

func DefaultConfig() *Config {
	c := &Config{}
	c.Context = context.Background()
	c.ImageFS = http.FS(os.DirFS("."))
	c.Styles = DefaultStyles()
	c.Options = make(map[goldrender.OptionName]interface{})

	return c
}

func (c *Config) AddDefaultNodeRenderers() {
	var nr NodeRenderer = &nodeRederFuncs{}

	c.NodeRenderers = append(c.NodeRenderers,
		util.Prioritized(nr, 1000),
	)
}
