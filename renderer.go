package pdf

import (
	"fmt"
	"io"
	"sync"

	"github.com/yuin/goldmark/ast"
	goldrender "github.com/yuin/goldmark/renderer"
)

// NewRreturns a new PDF Renderer with given options.
func New(options ...Option) goldrender.Renderer {
	config := DefaultConfig()
	for _, opt := range options {
		opt.SetConfig(config)
	}
	config.AddDefaultNodeRenderers()

	r := &renderer{
		config:               config,
		nodeRendererFuncsTmp: map[ast.NodeKind]NodeRendererFunc{},
	}

	return r
}

type renderer struct {
	config *Config

	nodeRendererFuncsTmp map[ast.NodeKind]NodeRendererFunc
	maxKind              int
	initSync             sync.Once
	nodeRendererFuncs    []NodeRendererFunc
}

// AddOptions has no effect on this renderer
// The method is to satisfy goldmark's Renderer interface
func (r *renderer) AddOptions(_ ...goldrender.Option) {
	// Nothing to add
}

// Satisfies the NodeRendererFuncRegisterer interface
// used to add NodeRenderers
func (r *renderer) Register(kind ast.NodeKind, v NodeRendererFunc) {
	r.nodeRendererFuncsTmp[kind] = v
	if int(kind) > r.maxKind {
		r.maxKind = int(kind)
	}
}

// Render renders the given AST node to the given writer.
func (r *renderer) Render(w io.Writer, source []byte, n ast.Node) error {
	r.initSync.Do(func() {
		// r.options = r.config.Options
		r.config.NodeRenderers.Sort()
		l := len(r.config.NodeRenderers)
		for i := l - 1; i >= 0; i-- {
			v := r.config.NodeRenderers[i]
			nr, _ := v.Value.(NodeRenderer)
			nr.RegisterFuncs(r)
		}
		r.nodeRendererFuncs = make([]NodeRendererFunc, r.maxKind+1)
		for kind, nr := range r.nodeRendererFuncsTmp {
			r.nodeRendererFuncs[kind] = nr
		}
	})

	pdf := r.config.PDF
	if pdf == nil {
		pdf = NewFpdf(r.config.Context, FpdfConfig{}, nil)
	}

	err := addStyleFonts(r.config.Context, pdf, r.config.Styles, r.config.FontsCache)
	if err != nil {
		return fmt.Errorf("could not load fonts: %w", err)
	}

	writer := &Writer{
		Pdf:         pdf,
		ImageFS:     r.config.ImageFS,
		Styles:      r.config.Styles,
		DebugWriter: r.config.TraceWriter,
		States:      states{stack: make([]*state, 0)},
	}

	mleft, _, _, _ := pdf.GetMargins()
	initcurrent := &state{
		containerType: ast.KindParagraph,
		listkind:      notlist,
		textStyle:     *r.config.Styles.Normal, leftMargin: mleft,
	}
	writer.States.push(initcurrent)

	err = ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		s := ast.WalkStatus(ast.WalkContinue)
		var err error
		f := r.nodeRendererFuncs[n.Kind()]
		if f != nil {
			s, err = f(writer, source, n, entering)
		}
		return s, err
	})
	if err != nil {
		return err
	}

	return pdf.Write(w)
}

func SetStyle(pdf PDF, s Style) {
	textR, textG, textB, _ := s.TextColor.RGBA()
	fillR, fillG, fillB, _ := s.FillColor.RGBA()

	pdf.SetFont(s.Font.Family, s.format, int(s.Size))
	pdf.SetTextColor(uint8(textR>>8), uint8(textG>>8), uint8(textB>>8))
	pdf.SetFillColor(uint8(fillR>>8), uint8(fillG>>8), uint8(fillB>>8))
	pdf.SetDrawColor(uint8(fillR>>8), uint8(fillG>>8), uint8(fillB>>8))
}
