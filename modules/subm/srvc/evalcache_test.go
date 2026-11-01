package srvc

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/stretchr/testify/require"
)

func TestEvalCacheConcurrentAccess(t *testing.T) {
	cache := newEvalCache()
	eval := domain.Eval{UUID: uuid.New(), Stage: domain.EvalStageWaiting}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			cache.set(eval)
		}()
		go func() {
			defer wg.Done()
			_, _ = cache.get(eval.UUID)
		}()
		go func() {
			defer wg.Done()
			cache.delete(eval.UUID)
		}()
	}
	wg.Wait()

	cache.set(eval)
	got, ok := cache.get(eval.UUID)
	require.True(t, ok)
	require.Equal(t, eval, got)

	cache.delete(eval.UUID)
	_, ok = cache.get(eval.UUID)
	require.False(t, ok)
}
