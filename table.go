package pdf

import (
	"bytes"
	"strings"

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

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
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

// CalculateTableOptimalColumnWidthsRowHeights returns per-column widths and
// per-body-row wrapped line counts sized to the table's content.
//
// Widths use a CSS-like min/max algorithm: each column gets its content's
// natural width when the table fits in the available page width, otherwise
// space is distributed proportionally between min (header/default) and max
// (longest cell) widths.
//
// Line counts are computed by wrapping each body cell at its final column
// width and taking the tallest cell in the row. Header rows always render
// single-line so they aren't included here. The returned slice is indexed
// by body-row position (matching curTableRow in the renderer).
func CalculateTableOptimalColumnWidthsRowHeights(w *Writer, tableData *TableData) ([]float64, []int) {
	if tableData == nil {
		return []float64{}, []int{}
	}

	// Headers define column count when present; otherwise fall back to the
	// first row. Subsequent rows with different cell counts are clamped to
	// numCols below to avoid out-of-range indexing.
	numCols := len(tableData.Headers)
	if numCols == 0 && len(tableData.Rows) > 0 {
		numCols = len(tableData.Rows[0])
	}

	if numCols == 0 {
		return []float64{}, []int{}
	}
	minWidths := make([]float64, numCols)
	maxWidths := make([]float64, numCols)

	pageWidth, _ := w.Pdf.GetPageSize()
	marginLeft, _, marginRight, _ := w.Pdf.GetMargins()
	availableWidth := pageWidth - marginLeft - marginRight

	// Measure each cell with the font it will actually render with, so the
	// line counts we predict here match what the renderer produces. If we
	// skipped this, the PDF's active font would be whatever ran last before
	// the table (e.g. a heading), and MeasureTextWidth/SplitText would give
	// the wrong character-per-line ratio — leading to over-tall rows.
	if w.Styles.THeader != nil {
		SetStyle(w.Pdf, *w.Styles.THeader)
	}
	if len(tableData.Headers) > 0 {
		for i, header := range tableData.Headers {
			minWidths[i] = w.Pdf.MeasureTextWidth(header) + (2 * w.Pdf.MeasureTextWidth("m"))
			maxWidths[i] = minWidths[i]
		}
	} else {
		defaultWidth := w.Pdf.MeasureTextWidth("mmmm")
		for i := 0; i < numCols; i++ {
			minWidths[i] = defaultWidth
			maxWidths[i] = defaultWidth
		}
	}

	if w.Styles.TBody != nil {
		SetStyle(w.Pdf, *w.Styles.TBody)
	}
	for _, row := range tableData.Rows {
		for i, cell := range row {
			if i < numCols {
				cellWidth := w.Pdf.MeasureTextWidth(strings.ReplaceAll(cell, "\n", " ")) + (2 * w.Pdf.MeasureTextWidth("m"))
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

	// Once widths are determined, count wrapped lines per row. Wrap at the
	// same effective width the renderer will use (column minus 1m of padding)
	// so the line count we predict here matches what gets drawn.
	mWidth := w.Pdf.MeasureTextWidth("m")
	finalMaxLines := make([]int, len(tableData.Rows))
	for i, row := range tableData.Rows {
		maxLines := 1
		for colIndex, cell := range row {
			if colIndex >= numCols {
				break
			}
			splitWidth := finalWidths[colIndex] - mWidth
			if splitWidth < 1 {
				splitWidth = finalWidths[colIndex]
			}
			lines := w.Pdf.SplitText(strings.ReplaceAll(cell, "\n", " "), splitWidth)
			maxLines = max(maxLines, len(lines))
		}
		finalMaxLines[i] = maxLines
	}

	return finalWidths, finalMaxLines
}
