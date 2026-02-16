// Package sagalog provides helpers for building SagaLog entries from
// an active OpenTelemetry span stored in a context.Context.
package sagalog

import (
	"context"
	"encoding/json"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// TraceInfo holds the OTel identifiers extracted from a context.
type TraceInfo struct {
	// TraceID is the W3C trace ID (32 lowercase hex chars).
	// Empty string if no active span is found in the context.
	TraceID string

	// SpanID is the W3C span ID (16 lowercase hex chars).
	SpanID string
}

// ExtractTraceInfo reads the active OpenTelemetry span from ctx and returns
// its trace_id and span_id as hex strings.
//
// How it works:
//  1. otelgrpc.NewServerHandler() (registered in each service's main.go)
//     extracts the W3C traceparent header from incoming gRPC metadata and
//     creates a server-side span, storing it in the context.
//  2. trace.SpanFromContext(ctx) retrieves that span.
//  3. span.SpanContext() gives us the TraceID and SpanID structs.
//  4. .IsValid() guards against the zero-value (no active span).
//
// If the context carries no active span (e.g. in unit tests), both fields
// are returned as empty strings — the caller should handle this gracefully.
func ExtractTraceInfo(ctx context.Context) TraceInfo {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()

	if !sc.IsValid() {
		return TraceInfo{}
	}

	return TraceInfo{
		TraceID: sc.TraceID().String(), // 32 hex chars, e.g. "4bf92f3577b34da6a3ce929d0e0e4736"
		SpanID:  sc.SpanID().String(),  // 16 hex chars, e.g. "00f067aa0ba902b7"
	}
}

// NewEntry is a convenience constructor that builds a SagaLog entry with
// the trace info automatically extracted from ctx.
//
// Usage in the orchestrator:
//
//	entry := sagalog.NewEntry(ctx, orderID, sagalog.StatusStepDone, "Inventory_Reservation_Step", "", nil)
//	_ = repo.Save(ctx, entry)
func NewEntry(
	ctx context.Context,
	sagaID string,
	status Status,
	currentStep string,
	payload string,
	errs []string,
) *SagaLog {
	ti := ExtractTraceInfo(ctx)

	errJSON := "[]"
	if len(errs) > 0 {
		if b, err := json.Marshal(errs); err == nil {
			errJSON = string(b)
		}
	}

	return &SagaLog{
		SagaID:        sagaID,
		Status:        status,
		CurrentStep:   currentStep,
		Payload:       payload,
		ErrorMessages: errJSON,
		TraceID:       ti.TraceID,
		SpanID:        ti.SpanID,
		UpdatedAt:     time.Now().UTC(),
	}
}
