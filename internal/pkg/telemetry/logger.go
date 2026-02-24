package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// ContextHandler is a custom slog.Handler that extracts TraceID and SpanID
// from the context and adds them as attributes to every log record.
type ContextHandler struct {
	slog.Handler
}

// Handle adds tracing context attributes before calling the underlying handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.HasTraceID() {
		r.AddAttrs(slog.String("trace_id", spanContext.TraceID().String()))
	}
	if spanContext.HasSpanID() {
		r.AddAttrs(slog.String("span_id", spanContext.SpanID().String()))
	}
	return h.Handler.Handle(ctx, r)
}

// NewContextHandler returns a new slog.Handler that decorates logs with tracing IDs.
func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: h}
}

// InitLogger initialises the global slog logger with a JSON handler decorated
// with tracing context.
func InitLogger() {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(NewContextHandler(handler))
	slog.SetDefault(logger)
}
