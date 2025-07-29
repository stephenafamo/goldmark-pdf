package pdf

import (
	"testing"
)

func TestCalculateOptimalColumnWidths(t *testing.T) {
	tests := []struct {
		name           string
		pageWidth      float64
		leftMargin     float64
		rightMargin    float64
		tableData      *TableData
		expectedWidths []float64
	}{
		{
			name:           "Empty table",
			pageWidth:      600,
			leftMargin:     50,
			rightMargin:    50,
			tableData:      nil,
			expectedWidths: []float64{},
		},
		{
			name:        "Table with headers only",
			pageWidth:   600,
			leftMargin:  50,
			rightMargin: 50,
			tableData: &TableData{
				Headers: []string{"Name", "Age", "City"},
				Rows:    [][]string{},
			},
			expectedWidths: []float64{30, 25, 30}, // 4*5+10, 3*5+10, 4*5+10
		},
		{
			name:        "Table with headers and short content",
			pageWidth:   600,
			leftMargin:  50,
			rightMargin: 50,
			tableData: &TableData{
				Headers: []string{"Name", "Age", "City"},
				Rows: [][]string{
					{"John", "30", "NYC"},
					{"Jane", "25", "LA"},
				},
			},
			expectedWidths: []float64{30, 25, 30}, // Headers are wider than content
		},
		{
			name:        "Table with headers and long content",
			pageWidth:   600,
			leftMargin:  50,
			rightMargin: 50,
			tableData: &TableData{
				Headers: []string{"Name", "Age", "City"},
				Rows: [][]string{
					{"Alexander Hamilton", "30", "New York City"},
					{"George Washington", "57", "Mount Vernon, Virginia"},
				},
			},
			expectedWidths: []float64{100, 25, 120}, // 18*5+10, 3*5+10, 22*5+10
		},
		{
			name:        "Table without headers",
			pageWidth:   600,
			leftMargin:  50,
			rightMargin: 50,
			tableData: &TableData{
				Headers: []string{},
				Rows: [][]string{
					{"John", "30", "NYC"},
					{"Jane", "25", "LA"},
				},
			},
			expectedWidths: []float64{30, 20, 25}, // 4*5+10, 2*5+10, 3*5+10
		},
		{
			name:        "Table with content exceeding page width",
			pageWidth:   300,
			leftMargin:  50,
			rightMargin: 50,
			tableData: &TableData{
				Headers: []string{"Very Long Header Name", "Another Long Header", "Third Long Header"},
				Rows: [][]string{
					{"This is a very long cell content", "Another long content here", "More long content"},
				},
			},
			expectedWidths: []float64{73.02, 66.67, 60.32}, // Distributed proportionally based on content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPdf := &MockPdf{
				pageWidth:   tt.pageWidth,
				pageHeight:  800,
				leftMargin:  tt.leftMargin,
				rightMargin: tt.rightMargin,
			}

			writer := &Writer{
				Pdf: mockPdf,
			}

			widths := CalculateOptimalColumnWidths(writer, tt.tableData)

			if len(widths) != len(tt.expectedWidths) {
				t.Errorf("Expected %d widths, got %d", len(tt.expectedWidths), len(widths))
				return
			}

			for i, w := range widths {
				// Allow small floating point differences
				if diff := w - tt.expectedWidths[i]; diff < -0.01 || diff > 0.01 {
					t.Errorf("Column %d: expected width %.2f, got %.2f", i, tt.expectedWidths[i], w)
				}
			}
		})
	}
}

func TestCalculateOptimalColumnWidths_EdgeCases(t *testing.T) {
	mockPdf := &MockPdf{
		pageWidth:   600,
		pageHeight:  800,
		leftMargin:  50,
		rightMargin: 50,
	}

	writer := &Writer{
		Pdf: mockPdf,
	}

	// Test with mismatched row lengths
	tableData := &TableData{
		Headers: []string{"Col1", "Col2", "Col3"},
		Rows: [][]string{
			{"A", "B"},           // Missing third column
			{"C", "D", "E", "F"}, // Extra column
		},
	}

	widths := CalculateOptimalColumnWidths(writer, tableData)

	// Should handle gracefully and return 3 columns based on headers
	if len(widths) != 3 {
		t.Errorf("Expected 3 widths for 3 headers, got %d", len(widths))
	}

	// Test proportional distribution
	tableData2 := &TableData{
		Headers: []string{"Small", "Medium Header", "Very Large Header Name"},
		Rows:    [][]string{},
	}

	widths2 := CalculateOptimalColumnWidths(writer, tableData2)

	// The third column should be wider than the second, which should be wider than the first
	if widths2[0] >= widths2[1] || widths2[1] >= widths2[2] {
		t.Errorf("Expected widths to be proportional to header lengths: %.2f, %.2f, %.2f",
			widths2[0], widths2[1], widths2[2])
	}
}
