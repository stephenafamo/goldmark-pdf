package pdf

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yuin/goldmark"
)

// captureHandler collects slog records for inspection in tests.
type captureHandler struct {
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

// writerWithStack returns a Writer whose state stack has the given depth
// (the LogDebug `depth` attribute is len(stack)-1).
func writerWithStack(depth int) *Writer {
	w := &Writer{}
	for i := 0; i <= depth; i++ {
		w.States.push(&state{})
	}
	return w
}

// TestWriterLogging_NilLoggerIsNoOp verifies that with a nil logger neither
// LogDebug nor LogWarn panics or emits anywhere.
func TestWriterLogging_NilLoggerIsNoOp(t *testing.T) {
	w := writerWithStack(0)
	w.LogDebug("debug source", "details")
	w.LogWarn("warn source", "details")
}

// TestWriterLogging_EmitsThroughLogger verifies that with a logger configured
// both helpers route records to it at the expected level.
func TestWriterLogging_EmitsThroughLogger(t *testing.T) {
	h := &captureHandler{}
	w := writerWithStack(0)
	w.Logger = slog.New(h)

	w.LogDebug("Paragraph", "entering")
	w.LogWarn("missing image", "details")

	if len(h.records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(h.records))
	}
	if h.records[0].Level != slog.LevelDebug || h.records[0].Message != "Paragraph" {
		t.Errorf("debug record = %v/%q, want Debug/Paragraph", h.records[0].Level, h.records[0].Message)
	}
	if h.records[1].Level != slog.LevelWarn || h.records[1].Message != "missing image" {
		t.Errorf("warn record = %v/%q, want Warn/missing image", h.records[1].Level, h.records[1].Message)
	}
}

// TestRender_MissingImageWarns drives a full markdown render that references
// an image not present in the configured FS, and asserts the renderer emits a
// Warn record naming the missing path.
func TestRender_MissingImageWarns(t *testing.T) {
	h := &captureHandler{}
	logger := slog.New(h)

	md := goldmark.New(
		goldmark.WithRenderer(New(
			WithPDF(&MockPdf{pageWidth: 600, pageHeight: 800, leftMargin: 50, rightMargin: 50}),
			WithImageFS(http.FS(fstest.MapFS{})), // empty FS so any lookup fails
			WithLogger(logger),
			// Inbuilt fonts skip the Google-fonts network fetch.
			WithHeadingFont(FontHelvetica),
			WithBodyFont(FontHelvetica),
			WithCodeFont(FontCourier),
		)),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte("![alt](nonexistent.png)\n"), &buf); err != nil {
		t.Fatalf("convert: %v", err)
	}

	var warn *slog.Record
	for i := range h.records {
		if h.records[i].Level == slog.LevelWarn {
			warn = &h.records[i]
			break
		}
	}
	if warn == nil {
		t.Fatalf("expected a Warn record for the missing image, got %d records", len(h.records))
	}
	if warn.Message != "Image (internal)" {
		t.Errorf("warn message = %q, want %q", warn.Message, "Image (internal)")
	}
	var msgAttr string
	warn.Attrs(func(a slog.Attr) bool {
		if a.Key == "msg" {
			msgAttr = a.Value.String()
			return false
		}
		return true
	})
	if !strings.Contains(msgAttr, "nonexistent.png") {
		t.Errorf("warn msg attr = %q, want it to mention the missing path", msgAttr)
	}
}
