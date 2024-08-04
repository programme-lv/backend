package tasks

import (
	"context"

	taskgen "github.com/programme-lv/backend/gen/tasks"
	"goa.design/clue/log"
)

// tasks service example implementation.
// The example methods log the requests and return zero values.
type taskssrvc struct{}

// NewTasks returns the tasks service implementation.
func NewTasks() taskgen.Service {
	return &taskssrvc{}
}

// List all tasks
func (s *taskssrvc) ListTasks(ctx context.Context) (res []*taskgen.Task, err error) {
	log.Printf(ctx, "tasks.listTasks")
	return
}

// Get a task by its ID
func (s *taskssrvc) GetTask(ctx context.Context, p *taskgen.GetTaskPayload) (res *taskgen.Task, err error) {
	res = &taskgen.Task{}
	log.Printf(ctx, "tasks.getTask")
	return
}
