package pdf

import "github.com/yuin/goldmark/ast"

type state struct {
	containerType  ast.NodeKind
	textStyle      Style
	leftMargin     float64
	firstParagraph bool

	// populated if node type is a list
	listkind   listType
	itemNumber int // only if an ordered list

	// populated if node type is a link
	destination string

	// populated if table cell
	isHeader bool
}

type states struct {
	stack []*state
}

func (s *states) push(c *state) {
	s.stack = append(s.stack, c)
}

func (s *states) pop() *state {
	var x *state
	x, s.stack = s.stack[len(s.stack)-1], s.stack[:len(s.stack)-1]
	return x
}

func (s *states) peek() *state {
	return s.stack[len(s.stack)-1]
}

func (s *states) parent() *state {
	return s.stack[len(s.stack)-2]
}
