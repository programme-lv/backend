package execsrvc

import (
	"sync"

	"github.com/google/uuid"
)

type InMemEvalRepo struct {
	lock  sync.Mutex
	evals map[uuid.UUID]Execution
}

func (m *InMemEvalRepo) Delete(evalUuid uuid.UUID) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.evals, evalUuid)
	return nil
}

func (m *InMemEvalRepo) Get(evalUuid uuid.UUID) (Execution, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	eval, ok := m.evals[evalUuid]
	if !ok {
		return Execution{}, ErrEvalNotFound()
	}
	return eval, nil
}

func (m *InMemEvalRepo) Save(eval Execution) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.evals[eval.UUID] = eval
	return nil
}

func NewInMemEvalRepo() *InMemEvalRepo {
	return &InMemEvalRepo{
		evals: make(map[uuid.UUID]Execution),
	}
}
