package gopdf

import (
	"reflect"
	"testing"
)

func TestSplitWords(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single word", "hello", []string{"hello"}},
		{"single space", " ", []string{" "}},
		{"two words", "hello world", []string{"hello", " ", "world"}},
		{"leading space", " hello", []string{" ", "hello"}},
		{"trailing space", "hello ", []string{"hello", " "}},
		{"multiple spaces between", "a   b", []string{"a", "   ", "b"}},
		{"three words", "one two three", []string{"one", " ", "two", " ", "three"}},
		{"all spaces", "   ", []string{"   "}},
		// Only ' ' splits — tabs and newlines stay inside chunks.
		{"tab is not a split", "a\tb", []string{"a\tb"}},
		{"newline is not a split", "a\nb", []string{"a\nb"}},
		{"unicode word", "héllo wörld", []string{"héllo", " ", "wörld"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitWords(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitWords(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// Exact (head, tail) outputs for the trivial cases.
func TestBreakAtWidth_ExactOutputs(t *testing.T) {
	impl := newFontedImpl(t)
	xWidth := impl.MeasureTextWidth("X")

	cases := []struct {
		name           string
		s              string
		w              float64
		wantHead, want string
	}{
		{"empty", "", 100, "", ""},
		{"zero width", "hello", 0, "", "hello"},
		{"negative width", "hello", -1, "", "hello"},
		{"fits entirely", "hi", 10000, "hi", ""},
		// Single overflowing rune still comes back as head — the "always make
		// progress" guarantee so callers don't loop forever.
		{"single rune overflow", "X", xWidth / 2, "X", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			head, tail := breakAtWidth(impl.GoPdf, tc.s, tc.w)
			if head != tc.wantHead || tail != tc.want {
				t.Errorf("got (%q, %q), want (%q, %q)", head, tail, tc.wantHead, tc.want)
			}
		})
	}
}

// For any positive width and non-empty input, head+tail must reconstruct the
// input exactly and head must be non-empty (forward progress).
func TestBreakAtWidth_NoRuneLoss(t *testing.T) {
	impl := newFontedImpl(t)

	inputs := []string{
		"the quick brown fox jumps over the lazy dog",
		"café naïve", // multi-byte runes — must not split mid-rune
		"aaaaaaaaaaaaaaaaaaaa",
	}

	for _, s := range inputs {
		full := impl.MeasureTextWidth(s)
		for _, frac := range []float64{0.1, 0.25, 0.5, 0.75, 0.9} {
			w := full * frac
			head, tail := breakAtWidth(impl.GoPdf, s, w)
			if head+tail != s {
				t.Errorf("s=%q frac=%v: head+tail=%q, want %q", s, frac, head+tail, s)
			}
			if head == "" {
				t.Errorf("s=%q frac=%v: head empty for positive width", s, frac)
			}
		}
	}
}

// When head has more than one rune, its width must actually fit the budget —
// the loop only emits an over-width head when forced to (single rune).
func TestBreakAtWidth_HeadFitsWidth(t *testing.T) {
	impl := newFontedImpl(t)
	const s = "aaaaaaaaaaaaaaaaaaaa"

	w := impl.MeasureTextWidth(s) / 3
	head, _ := breakAtWidth(impl.GoPdf, s, w)
	if headW := impl.MeasureTextWidth(head); headW > w && len([]rune(head)) > 1 {
		t.Errorf("head %q (width %v) exceeds budget %v with >1 rune", head, headW, w)
	}
}

// newFontedImpl returns an *Impl with a font loaded and selected, so
// MeasureTextWidth works. Skips when no system TTF is available.
func newFontedImpl(t *testing.T) *Impl {
	t.Helper()
	_, impl := newTestPDF(t)
	if err := impl.SetFont("DejaVu", "", 12); err != nil {
		t.Fatalf("SetFont: %v", err)
	}
	return impl
}
