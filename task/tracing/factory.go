package tracing

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/programme-lv/backend/task/srvc"
)

// NewTracedTaskService creates a new task service with tracing enabled
func NewTracedTaskService(repo srvc.TaskPgRepo) (srvc.TaskSrvcClient, *TracerProvider, error) {
	// Initialize the tracer provider
	tp, err := InitTracing("task-service")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	// Create the base task service
	baseService, err := srvc.NewTaskSrvc(repo)
	if err != nil {
		// Clean up tracer provider if service creation fails
		_ = tp.Shutdown(context.Background())
		return nil, nil, fmt.Errorf("failed to create task service: %w", err)
	}

	// Wrap the service with tracing
	tracedService := NewTaskSrvcTracer(baseService)

	slog.Info("Created traced task service")

	return tracedService, tp, nil
}

// ShutdownTracing gracefully shuts down the tracer provider
func ShutdownTracing(ctx context.Context, tp *TracerProvider) {
	if tp != nil {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down tracer provider", "error", err)
		} else {
			slog.Info("Tracer provider shut down successfully")
		}
	}
}
