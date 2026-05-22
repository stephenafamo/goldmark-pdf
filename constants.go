package pdf

type listType int

const (
	notlist listType = iota
	unordered
	ordered
	definition
)

var (
	// This slice of float64 contains the width of each cell
	// in the header of a table. These will be the widths used
	// in the table body as well.
	cellwidths []float64

	// Per-body-row wrapped line counts. Indexed by body-row position, matching
	// curTableRow in the renderer.
	cellMaxLines []int

	curTableCol int
	curTableRow int
	cellPadding float64

	fill = false
)

func (n listType) String() string {
	switch n {
	case notlist:
		return "Not a List"
	case unordered:
		return "Unordered"
	case ordered:
		return "Ordered"
	case definition:
		return "Definition"
	}
	return ""
}
