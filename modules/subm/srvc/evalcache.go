package srvc

import (
	"sync"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/modules/subm/domain"
)

// evalCache holds evaluations that are still being processed. Writes come
// from HTTP handlers and execution-event goroutines, reads from queries, so
// every access goes through the mutex.
type evalCache struct {
	mu    sync.Mutex
	evals map[uuid.UUID]domain.Eval
}

func newEvalCache() *evalCache {
	return &evalCache{evals: make(map[uuid.UUID]domain.Eval)}
}

func (c *evalCache) get(id uuid.UUID) (domain.Eval, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	eval, ok := c.evals[id]
	return eval, ok
}

func (c *evalCache) set(eval domain.Eval) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evals[eval.UUID] = eval
}

func (c *evalCache) delete(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.evals, id)
}
