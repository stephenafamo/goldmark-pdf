package pdf

type listType int

const (
	notlist listType = iota
	unordered
	ordered
	definition
)

// This slice of float64 contains the width of each cell
// in the header of a table. These will be the widths used
// in the table body as well.
var cellwidths []float64

var (
	curdatacell int
	fill        = false
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
