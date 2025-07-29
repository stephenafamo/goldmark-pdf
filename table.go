package pdf

import (
	"bytes"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// TableData stores the content of a table for width calculation
type TableData struct {
	Headers []string
	Rows    [][]string
}

var currentTableData *TableData

// ExtractCellText recursively extracts all text content from a cell
func ExtractCellText(source []byte, node ast.Node) string {
	var buf bytes.Buffer

	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindText:
			textNode := n.(*ast.Text)
			buf.Write(textNode.Segment.Value(source))
		case ast.KindString:
			stringNode := n.(*ast.String)
			buf.Write(stringNode.Value)
		}

		return ast.WalkContinue, nil
	})

	return buf.String()
}

// CollectTableData walks through table nodes to collect cell data
func CollectTableData(w *Writer, source []byte, node ast.Node) *TableData {
	tData := TableData{
		Headers: make([]string, 0),
		Rows:    make([][]string, 0),
	}

	if _, ok := node.(*east.Table); !ok {
		w.LogDebug("CollectTableData", "(not table)")
		return &tData
	}

	var extract func(w *Writer, source []byte, n ast.Node, inHeader bool)
	extract = func(w *Writer, source []byte, n ast.Node, inHeader bool) {
		switch n.Kind() {
		case east.KindTableHeader:
			for cell := n.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if cell.Kind() == east.KindTableCell {
					tData.Headers = append(tData.Headers, ExtractCellText(source, cell))
				}
			}
		case east.KindTableRow:
			var row []string
			for cell := n.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if cell.Kind() == east.KindTableCell {
					row = append(row, ExtractCellText(source, cell))
				}
			}
			tData.Rows = append(tData.Rows, row)
		default:
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				extract(w, source, child, inHeader)
			}
		}
	}

	// start extraction from table's children
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		extract(w, source, child, false)
	}

	return &tData
}

// CalculateOptimalColumnWidths based on content
func CalculateOptimalColumnWidths(w *Writer, tableData *TableData) []float64 {
	if tableData == nil {
		return []float64{}
	}

	// Determine number of columns from headers or first row
	numCols := len(tableData.Headers)
	if numCols == 0 && len(tableData.Rows) > 0 {
		numCols = len(tableData.Rows[0])
	}

	if numCols == 0 {
		return []float64{}
	}
	minWidths := make([]float64, numCols)
	maxWidths := make([]float64, numCols)

	// Get page dimensions
	pageWidth, _ := w.Pdf.GetPageSize()
	marginLeft, _, marginRight, _ := w.Pdf.GetMargins()
	availableWidth := pageWidth - marginLeft - marginRight

	// Calculate minimum widths based on headers
	if len(tableData.Headers) > 0 {
		for i, header := range tableData.Headers {
			minWidths[i] = w.Pdf.MeasureTextWidth(header) + (2 * w.Pdf.MeasureTextWidth("m"))
			maxWidths[i] = minWidths[i]
		}
	} else {
		// No headers, use a default minimum width
		defaultWidth := w.Pdf.MeasureTextWidth("mmmm")
		for i := 0; i < numCols; i++ {
			minWidths[i] = defaultWidth
			maxWidths[i] = defaultWidth
		}
	}

	// Calculate maximum widths based on content
	for _, row := range tableData.Rows {
		for i, cell := range row {
			if i < numCols {
				cellWidth := w.Pdf.MeasureTextWidth(cell) + (2 * w.Pdf.MeasureTextWidth("m"))
				if cellWidth > maxWidths[i] {
					maxWidths[i] = cellWidth
				}
			}
		}
	}

	// Calculate total minimum and maximum widths
	totalMin := 0.0
	totalMax := 0.0
	for i := 0; i < numCols; i++ {
		totalMin += minWidths[i]
		totalMax += maxWidths[i]
	}

	// Determine final column widths
	finalWidths := make([]float64, numCols)

	if totalMax <= availableWidth {
		// All columns can have their maximum width
		copy(finalWidths, maxWidths)
	} else if totalMin > availableWidth {
		// if minimum widths exceed available space, distribute proportionally
		ratio := availableWidth / totalMin
		for i := 0; i < numCols; i++ {
			finalWidths[i] = minWidths[i] * ratio
		}
	} else {
		// Distribute extra space proportionally
		extraSpace := availableWidth - totalMin
		maxExtraSpace := totalMax - totalMin

		for i := 0; i < numCols; i++ {
			proportion := (maxWidths[i] - minWidths[i]) / maxExtraSpace
			finalWidths[i] = minWidths[i] + (extraSpace * proportion)
		}
	}

	return finalWidths
}
