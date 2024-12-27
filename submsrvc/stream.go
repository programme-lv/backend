package submsrvc

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type EvalUpdate struct {
	SubmUuid uuid.UUID
	Eval     *Evaluation
}

func (s *SubmissionSrvc) broadcastSubmEvalUpdate(update *EvalUpdate) {
	s.listenerLock.Lock()
	defer s.listenerLock.Unlock()

	for _, listener := range s.submEvalUpdSubs {
		if listener.submUuid != update.SubmUuid {
			continue
		}
		select {
		case <-listener.ch:
			// Removed existing update
		default:
		}

		select {
		case listener.ch <- update:
		default:
			slog.Error("failed to send update to listener", "update", update)
		}
	}
}

func (s *SubmissionSrvc) broadcastNewSubmCreated(subm *Submission) {
	s.listenerLock.Lock()
	defer s.listenerLock.Unlock()

	for _, listener := range s.submCreatedSubs {
		select {
		case listener <- subm:
		default:
			slog.Error("failed to send update to listener", "update", subm)
		}
	}
}

func (s *SubmissionSrvc) ListenToLatestSubmEvalUpdate(ctx context.Context, submUuid uuid.UUID) (<-chan *EvalUpdate, error) {
	s.listenerLock.Lock()
	defer s.listenerLock.Unlock()

	ch := make(chan *EvalUpdate, 1)

	s.submEvalUpdSubs = append(s.submEvalUpdSubs, struct {
		submUuid uuid.UUID
		ch       chan *EvalUpdate
	}{submUuid, ch})

	go func() {
		<-ctx.Done()
		for i, listener := range s.submEvalUpdSubs {
			if listener.ch == ch {
				s.submEvalUpdSubs = append(s.submEvalUpdSubs[:i], s.submEvalUpdSubs[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch, nil
}

func (s *SubmissionSrvc) ListenToNewSubmCreated(ctx context.Context) (<-chan *Submission, error) {
	ch := make(chan *Submission, 1)
	s.submCreatedSubs = append(s.submCreatedSubs, ch)
	return ch, nil
}
