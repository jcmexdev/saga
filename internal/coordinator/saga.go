package coordinator

import (
	"context"
	"log/slog"

	"github.com/jcmexdev/ecommerce-sagas/internal/coordinator/sagalog"
)

// Step represents a single unit of work in the Saga.
// Each step must have a compensating action to undo its effects.
type Step interface {
	Name() string
	Execute(ctx context.Context) error
	Compensate(ctx context.Context) error
}

// Orchestrator manages the sequential execution of a collection of Steps.
// On failure, it triggers compensation in reverse order (LIFO).
//
// If a sagalog.Repository is provided, every state transition is persisted
// to the Saga Log so you can audit, debug, and recover sagas.
type Orchestrator struct {
	sagaID string
	steps  []Step
	log    sagalog.Repository // nil-safe: logging is skipped if nil
}

// NewOrchestrator creates a new Orchestrator.
//
//   - sagaID: the business identifier (typically the order ID). Used as the
//     primary key in the saga_logs table.
//   - repo: the saga log repository. Pass nil to disable logging (e.g. in tests).
func NewOrchestrator(sagaID string, steps []Step, repo sagalog.Repository) *Orchestrator {
	return &Orchestrator{
		sagaID: sagaID,
		steps:  steps,
		log:    repo,
	}
}

// Start runs the saga steps sequentially.
// If a step fails, it triggers the compensation of all previously successful
// steps in reverse order and returns the original error.
func (o *Orchestrator) Start(ctx context.Context) error {
	o.saveLog(ctx, sagalog.StatusStarted, "", "", nil)

	var completed []Step
	var errors []string

	for _, step := range o.steps {
		slog.InfoContext(ctx, "executing saga step", "saga_id", o.sagaID, "step", step.Name())

		if err := step.Execute(ctx); err != nil {
			slog.ErrorContext(ctx, "saga step failed, starting rollback",
				"saga_id", o.sagaID,
				"step", step.Name(),
				"error", err,
			)
			errors = append(errors, err.Error())
			o.saveLog(ctx, sagalog.StatusCompensating, step.Name(), "", errors)
			o.rollback(ctx, completed, errors)
			o.saveLog(ctx, sagalog.StatusFailed, step.Name(), "", errors)
			return err
		}

		slog.InfoContext(ctx, "saga step completed", "saga_id", o.sagaID, "step", step.Name())
		o.saveLog(ctx, sagalog.StatusStepDone, step.Name(), "", nil)
		completed = append(completed, step)
	}

	o.saveLog(ctx, sagalog.StatusCompleted, "", "", nil)
	slog.InfoContext(ctx, "saga completed successfully", "saga_id", o.sagaID)
	return nil
}

// rollback compensates all completed steps in reverse order (LIFO).
func (o *Orchestrator) rollback(ctx context.Context, steps []Step, errs []string) {
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		slog.InfoContext(ctx, "compensating saga step", "saga_id", o.sagaID, "step", step.Name())

		if err := step.Compensate(ctx); err != nil {
			slog.ErrorContext(ctx, "CRITICAL: compensation failed",
				"saga_id", o.sagaID,
				"step", step.Name(),
				"error", err,
			)
			errs = append(errs, "compensation of "+step.Name()+" failed: "+err.Error())
		}
	}
}

// saveLog persists a saga log entry. It is a no-op if no repository was provided.
// Errors are logged but never returned — a logging failure must never abort the saga.
func (o *Orchestrator) saveLog(ctx context.Context, status sagalog.Status, step, payload string, errs []string) {
	if o.log == nil {
		return
	}

	entry := sagalog.NewEntry(ctx, o.sagaID, status, step, payload, errs)

	if err := o.log.Save(ctx, entry); err != nil {
		// Non-fatal: the saga must continue even if the audit log fails.
		slog.WarnContext(ctx, "failed to save saga log entry",
			"saga_id", o.sagaID,
			"status", status,
			"error", err,
		)
	}
}
