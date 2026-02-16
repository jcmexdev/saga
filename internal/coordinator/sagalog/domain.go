// Package sagalog defines the domain types for the Saga Log pattern.
//
// A Saga Log is a durable audit trail of every state transition a saga goes
// through. It serves two purposes:
//
//  1. Observability: you can query the DB to see exactly where a saga is (or
//     was) and correlate it with a distributed trace via the trace_id field.
//
//  2. Recovery: on restart, the orchestrator can read the log and resume or
//     compensate sagas that were in-flight when the process crashed.
package sagalog

import "time"

// Status represents the lifecycle state of a saga execution.
type Status string

const (
	StatusStarted      Status = "STARTED"
	StatusStepDone     Status = "STEP_DONE"
	StatusCompleted    Status = "COMPLETED"
	StatusCompensating Status = "COMPENSATING"
	StatusFailed       Status = "FAILED"
)

// SagaLog is a single row in the saga_logs table.
// It captures a point-in-time snapshot of a saga execution.
type SagaLog struct {
	// SagaID is the unique identifier for this saga execution.
	// Typically the order ID so it can be joined with business data.
	SagaID string

	// Status is the current lifecycle state.
	Status Status

	// CurrentStep is the name of the step that was just executed or failed.
	CurrentStep string

	// Payload is the JSON-serialised input that started the saga.
	// Stored once at creation so the saga can be replayed from the log.
	Payload string

	// ErrorMessages accumulates failure details, one per failed step.
	// Stored as a JSON array: ["step X failed: ...", "compensation of Y failed: ..."]
	ErrorMessages string

	// TraceID is the W3C trace ID extracted from the OpenTelemetry span that
	// was active when this log entry was written. Allows you to jump directly
	// from a saga log row to the full distributed trace in Grafana/Tempo.
	TraceID string

	// SpanID is the specific span within the trace (useful for pinpointing
	// exactly which RPC call corresponds to this log entry).
	SpanID string

	// UpdatedAt is the wall-clock time of this log entry.
	UpdatedAt time.Time
}
