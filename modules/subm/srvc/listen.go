package srvc

import (
	"context"

	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/subm/domain"
)

func (s *submSrvc) SubscribeNewSubms(ctx context.Context) (<-chan domain.Subm, srvcerror.E) {
	ch := make(chan domain.Subm, 10)
	s.newSubmChListenerLock.Lock()
	s.newSubmListeners[ch] = struct{}{}
	s.newSubmChListenerLock.Unlock()
	go func() {
		<-ctx.Done()
		s.newSubmChListenerLock.Lock()
		delete(s.newSubmListeners, ch)
		s.newSubmChListenerLock.Unlock()
		close(ch)
	}()
	return ch, nil
}

func (s *submSrvc) SubscribeEvalUpds(ctx context.Context) (<-chan domain.Eval, srvcerror.E) {
	ch := make(chan domain.Eval, 10)
	s.newEvalUpdListenerLock.Lock()
	s.newEvalUpdListeners[ch] = struct{}{}
	s.newEvalUpdListenerLock.Unlock()
	go func() {
		<-ctx.Done()
		s.newEvalUpdListenerLock.Lock()
		delete(s.newEvalUpdListeners, ch)
		s.newEvalUpdListenerLock.Unlock()
		close(ch)
	}()
	return ch, nil
}

func (s *submSrvc) broadcastEvalUpdate(eval domain.Eval) {
	s.newEvalUpdListenerLock.Lock()
	defer s.newEvalUpdListenerLock.Unlock()
	for ch := range s.newEvalUpdListeners {
		select {
		case ch <- eval:
		default:
			<-ch
			ch <- eval
		}
	}
}

func (s *submSrvc) broadcastSubmCreated(subm domain.Subm) {
	s.newSubmChListenerLock.Lock()
	defer s.newSubmChListenerLock.Unlock()
	for ch := range s.newSubmListeners {
		select {
		case ch <- subm:
		default:
			<-ch
			ch <- subm
		}
	}
}
