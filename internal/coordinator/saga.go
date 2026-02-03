package coordinator

import (
	"context"
	"log"
)

// Step represents a single unit of work in the Saga.
// Each step must have a compensating action to undo its effects.
type Step interface {
	Name() string
	Execute(ctx context.Context) error
	Compensate(ctx context.Context) error
}

// Orchestrator manages the execution of a collection of Steps.
type Orchestrator struct {
	steps []Step
}

func NewOrchestrator(steps []Step) *Orchestrator {
	return &Orchestrator{steps: steps}
}

// Start runs the saga steps sequentially.
// If a step fails, it triggers the compensation of all previously successful steps.
func (o *Orchestrator) Start(ctx context.Context) error {
	var successfulSteps []Step

	for _, step := range o.steps {
		log.Printf("Executing step: %s", step.Name())
		if err := step.Execute(ctx); err != nil {
			log.Printf("Step %s failed: %v. Starting rollback...", step.Name(), err)
			o.rollback(ctx, successfulSteps)
			return err
		}
		// Track successful step for potential compensation (LIFO)
		successfulSteps = append(successfulSteps, step)
	}

	log.Println("Saga completed successfully")
	return nil
}

func (o *Orchestrator) rollback(ctx context.Context, steps []Step) {
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		log.Printf("Compensating step: %s", step.Name())
		if err := step.Compensate(ctx); err != nil {
			log.Printf("CRITICAL: Failed to compensate step %s: %v", step.Name(), err)
		}
	}
}
