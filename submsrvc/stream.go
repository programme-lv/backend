package submsrvc

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type EvalUpdate struct {
	SubmUuid uuid.UUID
	Eval     Evaluation
}

func (s *SubmissionSrvc) broadcastSubmEvalUpdate(update *EvalUpdate) {
	s.listenerLock.Lock()
	defer s.listenerLock.Unlock()

	for _, listener := range s.submUuidEvalUpdSubs {
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

	for _, listener := range s.submListEvalUpdSubs {
		// TODO: in the future, check whether the submission is part of the public submission list.
		// this will only be necessary when we want to restrict users from seeing submission before they have solved the task themselves.
		// for now, we just broadcast to all listeners.
		// We will need to somehow know that 1) the user receiving the update has solved the task themselves, and
		// 2) the submission is not part of an ongoing contest where standings are not public.
		// This will mean adding a new field to listener, being the user's uuid, and checking that the user has solved the task themselves.
		// We will need to implement what's known as a "restricted" evaluation update, which is a bit more complex.
		// Empty channel into slice
		var updates []*EvalUpdate
		for {
			select {
			case u := <-listener:
				updates = append(updates, u)
			default:
				goto done
			}
		}
	done:
		// Remove updates with same UUID
		filtered := make([]*EvalUpdate, 0, len(updates))
		for _, u := range updates {
			if u.SubmUuid != update.SubmUuid {
				filtered = append(filtered, u)
			}
		}

		// Add current update
		filtered = append(filtered, update)

		// Get channel capacity
		cap := cap(listener)

		// Keep only most recent updates that fit in channel
		if len(filtered) > cap {
			filtered = filtered[len(filtered)-cap:]
		}

		// Send updates back to channel
		for _, u := range filtered {
			select {
			case listener <- u:
			default:
				s.logger.Error("failed to send update to listener", "update", u)
			}
		}
	}
}

func (s *SubmissionSrvc) broadcastNewSubmCreated(subm Submission) {
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

func (s *SubmissionSrvc) ListenToSubmListEvalUpdates(ctx context.Context) (<-chan *EvalUpdate, error) {
	s.listenerLock.Lock()
	defer s.listenerLock.Unlock()

	ch := make(chan *EvalUpdate, 100)
	s.submListEvalUpdSubs = append(s.submListEvalUpdSubs, ch)
	return ch, nil
}

func (s *SubmissionSrvc) ListenToLatestSubmEvalUpdate(ctx context.Context, submUuid uuid.UUID) (<-chan *EvalUpdate, error) {
	s.listenerLock.Lock()
	defer s.listenerLock.Unlock()

	ch := make(chan *EvalUpdate, 1)

	s.submUuidEvalUpdSubs = append(s.submUuidEvalUpdSubs, struct {
		submUuid uuid.UUID
		ch       chan *EvalUpdate
	}{submUuid, ch})

	go func() {
		<-ctx.Done()
		for i, listener := range s.submUuidEvalUpdSubs {
			if listener.ch == ch {
				s.submUuidEvalUpdSubs = append(s.submUuidEvalUpdSubs[:i], s.submUuidEvalUpdSubs[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch, nil
}

func (s *SubmissionSrvc) ListenToNewSubmCreated(ctx context.Context) (<-chan Submission, error) {
	ch := make(chan Submission, 1)
	s.submCreatedSubs = append(s.submCreatedSubs, ch)
	return ch, nil
}
