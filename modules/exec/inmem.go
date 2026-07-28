package exec

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type InMemExecRepo struct {
	mu         sync.Mutex
	executions map[uuid.UUID]*Execution
}

var _ ExecRepo = &InMemExecRepo{}

func NewInMemExecRepo() *InMemExecRepo {
	return &InMemExecRepo{
		executions: make(map[uuid.UUID]*Execution),
	}
}

// Get implements [ExecRepo].
func (i *InMemExecRepo) Get(ctx context.Context, id uuid.UUID) (*Execution, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	exec, ok := i.executions[id]
	if !ok {
		return nil, fmt.Errorf("execution not found")
	}
	return exec, nil
}

// Save implements [ExecRepo].
func (i *InMemExecRepo) Save(ctx context.Context, exec *Execution) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.executions[exec.UUID] = exec
	return nil
}

var _ ExecRepo = &InMemExecRepo{}
