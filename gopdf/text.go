package gopdf

import (
	"strings"

	"github.com/signintech/gopdf"
)

// splitWords splits text into alternating non-space and whitespace runs.
// Each run is one chunk; the renderer wraps between chunks.
func splitWords(text string) []string {
	var chunks []string
	var buf strings.Builder
	inSpace := false
	for i, r := range text {
		if i == 0 {
			inSpace = r == ' '
			buf.WriteRune(r)
			continue
		}
		isSpace := r == ' '
		if isSpace != inSpace {
			chunks = append(chunks, buf.String())
			buf.Reset()
			inSpace = isSpace
		}
		buf.WriteRune(r)
	}
	if buf.Len() > 0 {
		chunks = append(chunks, buf.String())
	}
	return chunks
}

// breakAtWidth returns the largest rune-aligned prefix of s that fits in w,
// plus the remainder. Always returns at least one rune in head (or, when w is
// non-positive, the whole string in tail) so callers won't loop forever.
func breakAtWidth(gpdf *gopdf.GoPdf, s string, w float64) (head, tail string) {
	runes := []rune(s)
	if len(runes) == 0 {
		return "", ""
	}
	if w <= 0 {
		return "", s
	}

	var line []rune
	for i, r := range runes {
		candidate := append(line, r) //nolint:gocritic // we reassign line below; aliasing is intentional
		cw, _ := gpdf.MeasureTextWidth(string(candidate))
		if cw > w && len(line) > 0 {
			return string(line), string(runes[i:])
		}
		line = candidate
	}
	return string(line), ""
}
