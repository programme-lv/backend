package exec

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	testerapi "github.com/programme-lv/tester/api"
	"github.com/stretchr/testify/require"
)

type publishRecorder struct {
	mu   sync.Mutex
	msgs []*nats.Msg
	ch   chan *nats.Msg
}

func newPublishRecorder() *publishRecorder {
	return &publishRecorder{ch: make(chan *nats.Msg, 16)}
}

func (r *publishRecorder) publish(msg *nats.Msg) error {
	r.mu.Lock()
	r.msgs = append(r.msgs, msg)
	r.mu.Unlock()
	r.ch <- msg
	return nil
}

func (r *publishRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.msgs)
}

func (r *publishRecorder) waitForPublish(t *testing.T) *nats.Msg {
	t.Helper()
	select {
	case msg := <-r.ch:
		return msg
	case <-time.After(time.Second):
		t.Fatal("no job published")
		return nil
	}
}

func newWatchdogTestSrvc(t *testing.T) (*execSrvc, *publishRecorder) {
	t.Helper()
	srvc := NewExecSrvc(t.Context(), NewInMemExecRepo(), &nats.Conn{}, nil)
	srvc.publishRetryDelay = time.Millisecond
	srvc.watchdogInterval = time.Millisecond
	rec := newPublishRecorder()
	srvc.publishJob = rec.publish
	return srvc, rec
}

func enqueueWatchdogTestJob(t *testing.T, srvc *execSrvc, rec *publishRecorder) uuid.UUID {
	t.Helper()
	content := "print(1)"
	execUUID := uuid.New()
	err := srvc.Enqueue(t.Context(), execUUID, "print(1)", "python3.13", []TestFile{{
		InContent: &content, AnsContent: &content,
	}}, TestingParams{CpuMs: 1000, MemKiB: 1024})
	require.NoError(t, err)
	require.Equal(t, 1, rec.count())
	return execUUID
}

func noRespondersMsg(srvc *execSrvc, execUUID uuid.UUID) *nats.Msg {
	msg := nats.NewMsg(srvc.resultSubject(execUUID))
	msg.Header.Set("Status", noRespondersStatus)
	return msg
}

func TestNoRespondersRepublishesJob(t *testing.T) {
	srvc, rec := newWatchdogTestSrvc(t)
	execUUID := enqueueWatchdogTestJob(t, srvc, rec)
	first := rec.waitForPublish(t)

	srvc.handleResultMessage(t.Context(), noRespondersMsg(srvc, execUUID))

	second := rec.waitForPublish(t)
	require.Equal(t, first.Subject, second.Subject)
	require.Equal(t, srvc.resultSubject(execUUID), second.Reply)
	require.Equal(t, first.Data, second.Data)
}

func TestNoRespondersEndsExecutionAfterAttempts(t *testing.T) {
	srvc, rec := newWatchdogTestSrvc(t)
	srvc.maxPublishAttempts = 2
	execUUID := enqueueWatchdogTestJob(t, srvc, rec)
	events := listenEvents(t, srvc, execUUID)
	rec.waitForPublish(t)

	srvc.handleResultMessage(t.Context(), noRespondersMsg(srvc, execUUID))
	rec.waitForPublish(t)
	srvc.handleResultMessage(t.Context(), noRespondersMsg(srvc, execUUID))

	execution, err := srvc.Get(t.Context(), execUUID)
	require.NoError(t, err)
	require.Equal(t, StageInternalError, execution.Stage)
	require.NotNil(t, execution.ErrorMsg)
	require.Equal(t, testerUnavailableMsg, *execution.ErrorMsg)
	require.IsType(t, TesterUnavailable{}, lastEvent(t, events))
}

func TestWatchdogRepublishesJobWithoutEvents(t *testing.T) {
	srvc, rec := newWatchdogTestSrvc(t)
	srvc.maxPublishAttempts = 3
	execUUID := enqueueWatchdogTestJob(t, srvc, rec)
	rec.waitForPublish(t)

	future := time.Now().Add(time.Hour)
	srvc.sweepExecutions(t.Context(), future)
	rec.waitForPublish(t)
	srvc.sweepExecutions(t.Context(), future)
	rec.waitForPublish(t)
	srvc.sweepExecutions(t.Context(), future)

	execution, err := srvc.Get(t.Context(), execUUID)
	require.NoError(t, err)
	require.Equal(t, StageInternalError, execution.Stage)
	require.NotNil(t, execution.ErrorMsg)
	require.Equal(t, testerUnavailableMsg, *execution.ErrorMsg)
}

func TestWatchdogEndsStalledExecution(t *testing.T) {
	srvc, rec := newWatchdogTestSrvc(t)
	execUUID := enqueueWatchdogTestJob(t, srvc, rec)
	events := listenEvents(t, srvc, execUUID)
	rec.waitForPublish(t)

	start := testerapi.NewStartJob(execUUID.String(), "test machine")
	data, err := json.Marshal(start)
	require.NoError(t, err)
	srvc.handleResultMessage(t.Context(), &nats.Msg{
		Subject: srvc.resultSubject(execUUID),
		Data:    data,
	})

	srvc.sweepExecutions(context.Background(), time.Now().Add(time.Hour))

	execution, err := srvc.Get(t.Context(), execUUID)
	require.NoError(t, err)
	require.Equal(t, StageInternalError, execution.Stage)
	require.NotNil(t, execution.ErrorMsg)
	require.Equal(t, testerTimeoutMsg, *execution.ErrorMsg)
	require.IsType(t, InternalServerError{}, lastEvent(t, events))
}

func listenEvents(t *testing.T, srvc *execSrvc, execUUID uuid.UUID) <-chan Event {
	t.Helper()
	events, err := srvc.Listen(t.Context(), execUUID)
	require.NoError(t, err)
	return events
}

func lastEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()
	var all []Event
	for event := range events {
		all = append(all, event)
	}
	require.NotEmpty(t, all)
	return all[len(all)-1]
}

func TestEvalUUIDFromSubject(t *testing.T) {
	id := uuid.New()
	parsed, ok := evalUUIDFromSubject("_INBOX.abc." + id.String())
	require.True(t, ok)
	require.Equal(t, id, parsed)

	_, ok = evalUUIDFromSubject("_INBOX.abc")
	require.False(t, ok)

	_, ok = evalUUIDFromSubject("_INBOX.abc.not-a-uuid")
	require.False(t, ok)
}
