package pdf

import "io"

// MockPdf implements the PDF interface for testing
type MockPdf struct {
	pageWidth   float64
	pageHeight  float64
	leftMargin  float64
	rightMargin float64
}

func (m *MockPdf) AddInternalLink(anchor string) {}
func (m *MockPdf) AddPage()                      {}
func (m *MockPdf) BR(h float64)                  {}
func (m *MockPdf) CellFormat(w, h float64, txt, border string, ln int, align string, fill bool, link int, linkStr string) {
}
func (m *MockPdf) GetMargins() (left, top, right, bottom float64) {
	return m.leftMargin, 10, m.rightMargin, 10
}
func (m *MockPdf) GetPageSize() (width, height float64) {
	return m.pageWidth, m.pageHeight
}
func (m *MockPdf) GetX() float64               { return 0 }
func (m *MockPdf) GetY() float64               { return 0 }
func (m *MockPdf) Line(x1, x2, y1, y2 float64) {}
func (m *MockPdf) MeasureTextWidth(s string) float64 {
	// Simple approximation: each character is 5 units wide
	return float64(len(s)) * 5.0
}
func (m *MockPdf) RegisterImage(id, format string, src io.Reader)  {}
func (m *MockPdf) SetDrawColor(r, g, b uint8)                      {}
func (m *MockPdf) SetFillColor(r, g, b uint8)                      {}
func (m *MockPdf) SetFont(family, style string, size int) error    { return nil }
func (m *MockPdf) AddFont(family, style string, data []byte) error { return nil }
func (m *MockPdf) SetLineWidth(width float64)                      {}
func (m *MockPdf) SetMarginLeft(margin float64)                    {}
func (m *MockPdf) SetMarginRight(margin float64)                   {}
func (m *MockPdf) SetTextColor(r, g, b uint8)                      {}
func (m *MockPdf) SetX(x float64)                                  {}
func (m *MockPdf) SetY(y float64)                                  {}
func (m *MockPdf) SplitText(txt string, w float64) []string {
	// Simple implementation for testing
	if m.MeasureTextWidth(txt) <= w {
		return []string{txt}
	}
	// Split into chunks
	charsPerLine := int(w / 5.0) // 5 units per character
	var lines []string
	for i := 0; i < len(txt); i += charsPerLine {
		end := i + charsPerLine
		if end > len(txt) {
			end = len(txt)
		}
		lines = append(lines, txt[i:end])
	}
	return lines
}
func (m *MockPdf) UseImage(id string, x, y, w, h float64)                           {}
func (m *MockPdf) Write(w io.Writer) error                                          { return nil }
func (m *MockPdf) WriteText(h float64, txtStr string)                               {}
func (m *MockPdf) WriteInternalLink(lineHeight float64, text string, anchor string) {}
func (m *MockPdf) WriteExternalLink(lineHeight float64, text, destination string)   {}
