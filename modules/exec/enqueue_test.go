package exec

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

func TestEnqueuePublishFailureIsNotObservable(t *testing.T) {
	srvc := NewExecSrvc(t.Context(), NewInMemExecRepo(), &nats.Conn{}, nil)
	publishing := make(chan struct{})
	release := make(chan struct{})
	srvc.publishJob = func(*nats.Msg) error {
		close(publishing)
		<-release
		return errors.New("publish unavailable")
	}

	id := uuid.New()
	content := "test"
	enqueueDone := make(chan error, 1)
	go func() {
		enqueueDone <- srvc.Enqueue(t.Context(), id, "print(1)", "python3.13", []TestFile{{
			InContent: &content, AnsContent: &content,
		}}, TestingParams{CpuMs: 1000, MemKiB: 1024})
	}()
	<-publishing

	listenDone := make(chan error, 1)
	go func() {
		_, err := srvc.Listen(t.Context(), id)
		listenDone <- err
	}()
	getDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := srvc.Get(ctx, id)
		getDone <- err
	}()

	requireBlocked(t, listenDone)
	requireBlocked(t, getDone)
	close(release)
	require.Error(t, <-enqueueDone)
	require.Error(t, <-listenDone)
	require.Error(t, <-getDone)
	assertExecutionStateAbsent(t, srvc, id)
}

func requireBlocked(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("observer completed before publish: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
}

func assertExecutionStateAbsent(t *testing.T, srvc *execSrvc, id uuid.UUID) {
	t.Helper()
	srvc.mu.Lock()
	defer srvc.mu.Unlock()
	require.NotContains(t, srvc.notifiers, id)
	require.NotContains(t, srvc.organizers, id)
	require.NotContains(t, srvc.executions, id)
	require.NotContains(t, srvc.fileHashes, id)
	_, exists := srvc.execWg.Load(id)
	require.False(t, exists)
}
