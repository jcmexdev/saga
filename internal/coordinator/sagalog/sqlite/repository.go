// Package sqlite provides a SQLite-backed implementation of sagalog.Repository.
//
// WAL mode is enabled on Open so that readers never block writers and vice
// versa — important because the saga goroutine writes while the HTTP handler
// may be reading (e.g. for a status endpoint).
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jcmexdev/ecommerce-sagas/internal/coordinator/sagalog"

	// Register the pure-Go SQLite driver.
	// We use modernc.org/sqlite instead of mattn/go-sqlite3 to avoid CGO
	// requirements, making it easier to build and run in Docker (Alpine).
	_ "modernc.org/sqlite"
)

// schema is the DDL executed once on startup.
// The table is append-only: each row is an immutable event in the saga's
// lifecycle. Querying MAX(updated_at) per saga_id gives the current state.
const schema = `
CREATE TABLE IF NOT EXISTS saga_logs (
    -- Surrogate primary key — auto-incremented by SQLite.
    id              INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Business identifier: typically the order ID.
    -- Not UNIQUE because multiple rows exist per saga (one per transition).
    saga_id         TEXT        NOT NULL,

    -- Lifecycle state at the time this row was written.
    status          TEXT        NOT NULL,

    -- Name of the step that just executed (e.g. "Inventory_Reservation_Step").
    current_step    TEXT        NOT NULL DEFAULT '',

    -- JSON payload that started the saga. Written once on STARTED, NULL after.
    payload         TEXT,

    -- JSON array of error strings accumulated during failure/compensation.
    error_messages  TEXT        NOT NULL DEFAULT '[]',

    -- W3C trace_id (32 hex chars) from the active OTel span.
    -- Allows jumping from this row directly to the trace in Grafana/Tempo.
    trace_id        TEXT        NOT NULL DEFAULT '',

    -- W3C span_id (16 hex chars) — pinpoints the exact RPC within the trace.
    span_id         TEXT        NOT NULL DEFAULT '',

    -- Wall-clock timestamp of this event (RFC3339 stored as TEXT, SQLite idiom).
    updated_at      TEXT        NOT NULL
);

-- Index for the most common query: "give me all events for saga X in order".
CREATE INDEX IF NOT EXISTS idx_saga_logs_saga_id ON saga_logs(saga_id, updated_at);

-- Index for the observability query: "find the saga for trace Y".
CREATE INDEX IF NOT EXISTS idx_saga_logs_trace_id ON saga_logs(trace_id);
`

// Repository is the SQLite implementation of sagalog.Repository.
type Repository struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at the given path and applies
// the schema. WAL mode is enabled for better concurrent read/write performance.
//
//	repo, err := sqlite.Open("./data/saga.db")
func Open(path string) (*Repository, error) {
	// The pure-Go driver uses _pragma query parameters to configure connection state.
	// WAL enables concurrent readers. foreign_keys=on enforces integrity (good practice).
	// busy_timeout waits for locks instead of failing immediately.
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)", path)

	// Use "sqlite", not "sqlite3" for the modernc driver.
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", path, err)
	}

	// SQLite performs best with a single writer connection.
	// Readers can use additional connections from the pool.
	db.SetMaxOpenConns(1)

	if err := applySchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Repository{db: db}, nil
}

// Close releases the database connection. Call it with defer in main().
func (r *Repository) Close() error {
	return r.db.Close()
}

// Save inserts a new saga log entry. It is safe to call concurrently.
func (r *Repository) Save(ctx context.Context, entry *sagalog.SagaLog) error {
	const q = `
		INSERT INTO saga_logs
			(saga_id, status, current_step, payload, error_messages, trace_id, span_id, updated_at)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, q,
		entry.SagaID,
		string(entry.Status),
		entry.CurrentStep,
		nullableString(entry.Payload),
		entry.ErrorMessages,
		entry.TraceID,
		entry.SpanID,
		entry.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z"),
	)
	if err != nil {
		return fmt.Errorf("sqlite: save saga log for %q: %w", entry.SagaID, err)
	}
	return nil
}

// GetLatest returns the most recent log entry for a given saga ID.
// Useful for a status endpoint or for recovery on restart.
func (r *Repository) GetLatest(ctx context.Context, sagaID string) (*sagalog.SagaLog, error) {
	const q = `
		SELECT saga_id, status, current_step, COALESCE(payload,''), error_messages,
		       trace_id, span_id, updated_at
		FROM   saga_logs
		WHERE  saga_id = ?
		ORDER  BY updated_at DESC, id DESC
		LIMIT  1`

	row := r.db.QueryRowContext(ctx, q, sagaID)

	var entry sagalog.SagaLog
	var updatedAt string
	err := row.Scan(
		&entry.SagaID,
		&entry.Status,
		&entry.CurrentStep,
		&entry.Payload,
		&entry.ErrorMessages,
		&entry.TraceID,
		&entry.SpanID,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sqlite: saga %q not found", sagaID)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: get latest for %q: %w", sagaID, err)
	}

	entry.UpdatedAt, err = parseRFC3339(updatedAt)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// applySchema runs the DDL statements once. Idempotent due to IF NOT EXISTS.
func applySchema(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("sqlite: apply schema: %w", err)
	}
	return nil
}

// nullableString returns nil for empty strings so SQLite stores NULL instead
// of an empty TEXT — keeps the payload column clean on non-STARTED rows.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
