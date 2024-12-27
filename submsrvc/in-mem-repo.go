package submsrvc

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type inMemRepo struct {
	mu    sync.RWMutex
	subms map[uuid.UUID]Submission
}

// List implements submRepo.
func (r *inMemRepo) List(ctx context.Context) ([]Submission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	subms := make([]Submission, 0, len(r.subms))
	for _, subm := range r.subms {
		subms = append(subms, subm)
	}
	return subms, nil
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
	return nil, fmt.Errorf("submission not found")
}

var _ submRepo = &inMemRepo{}
