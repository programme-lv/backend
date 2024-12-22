package evalsrvc

import (
	"sync"

	"github.com/google/uuid"
)

type InMemEvalRepo struct {
	lock  sync.Mutex
	evals map[uuid.UUID]Evaluation
}

func (m *InMemEvalRepo) Delete(evalUuid uuid.UUID) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.evals, evalUuid)
	return nil
}

func (m *InMemEvalRepo) Get(evalUuid uuid.UUID) (Evaluation, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	eval, ok := m.evals[evalUuid]
	if !ok {
		return Evaluation{}, ErrEvalNotFound()
	}
	return eval, nil
}

func (m *InMemEvalRepo) Save(eval Evaluation) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.evals[eval.UUID] = eval
	return nil
}

func NewInMemEvalRepo() *InMemEvalRepo {
	return &InMemEvalRepo{
		evals: make(map[uuid.UUID]Evaluation),
	}
}
