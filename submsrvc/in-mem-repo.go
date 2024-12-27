package submsrvc

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type inMemRepo struct {
	mu    sync.RWMutex
	subms map[uuid.UUID]Submission
}

func newInMemRepo() *inMemRepo {
	return &inMemRepo{
		subms: make(map[uuid.UUID]Submission),
	}
}

// Store implements submRepo
func (r *inMemRepo) Store(ctx context.Context, subm Submission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subms[subm.UUID] = subm
	return nil
}

// Get implements submRepo
func (r *inMemRepo) Get(ctx context.Context, uuid uuid.UUID) (*Submission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if subm, ok := r.subms[uuid]; ok {
		return &subm, nil
	}
	return nil, nil
}
