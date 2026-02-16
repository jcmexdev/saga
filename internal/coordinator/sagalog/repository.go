package sagalog

import "context"

// Repository is the port (interface) for persisting saga log entries.
// The coordinator depends on this abstraction, not on SQLite directly,
// so you can swap the implementation for Postgres, in-memory (tests), etc.
type Repository interface {
	// Save persists a new log entry. Each call appends a row; the table is
	// an append-only audit log, not an upsert.
	Save(ctx context.Context, entry *SagaLog) error
}
