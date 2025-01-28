package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submsrvc/submadapter"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
)

type SubmSrvcClient interface {
	SubmitSol(ctx context.Context, p submcmd.SubmitSolParams) error
	ReEvalSubm(ctx context.Context, p submcmd.ReEvalSubmParams) error
	GetSubm(ctx context.Context, uuid uuid.UUID) (subm.Subm, error)
	ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]subm.Subm, error)
	GetEval(ctx context.Context, uuid uuid.UUID) (subm.Eval, error)
	SubsNewSubm(ctx context.Context) (<-chan subm.Subm, error)
	SubsEvalUpd(ctx context.Context) (<-chan subm.Eval, error)
}

type SubmSrvc struct {
	userSrvc submadapter.UserSrvcFacade
	taskSrvc submadapter.TaskSrvcFacade
	execSrvc submadapter.ExecSrvcFacade
	submRepo submadapter.SubmRepo
	evalRepo submadapter.EvalRepo

	newSubmChListenerLock sync.Mutex
	newSubmListeners      map[chan<- subm.Subm]struct{}

	newEvalUpdListenerLock sync.Mutex
	newEvalUpdListeners    map[chan<- subm.Eval]struct{}
}

func NewSubmSrvc(
	userSrvc submadapter.UserSrvcFacade,
	taskSrvc submadapter.TaskSrvcFacade,
	execSrvc submadapter.ExecSrvcFacade,
	submRepo submadapter.SubmRepo,
	evalRepo submadapter.EvalRepo,
) SubmSrvcClient {
	return &SubmSrvc{
		userSrvc: userSrvc,
		taskSrvc: taskSrvc,
		execSrvc: execSrvc,
		submRepo: submRepo,
		evalRepo: evalRepo,

		newSubmListeners:    make(map[chan<- subm.Subm]struct{}),
		newEvalUpdListeners: make(map[chan<- subm.Eval]struct{}),
	}
}

func (s *SubmSrvc) SubmitSol(ctx context.Context, p submcmd.SubmitSolParams) error {
	submitSolCmd := submcmd.SubmitSolCmdHandler{
		DoesUserExist: func(ctx context.Context, uuid uuid.UUID) (bool, error) {
			user, err := s.userSrvc.GetUserByUUID(ctx, uuid)
			if err != nil {
				return false, err
			}
			return user.UUID == uuid, nil
		},
		GetTask:   s.taskSrvc.GetTask,
		StoreSubm: s.submRepo.StoreSubm,
		StoreEval: s.evalRepo.StoreEval,
		BcastSubmCreated: func(subm subm.Subm) {
			slog.Info("submitted solution", "subm", subm)
			s.newSubmChListenerLock.Lock()
			for ch := range s.newSubmListeners {
				ch <- subm
			}
			s.newSubmChListenerLock.Unlock()
		},
		EnqueueEvalExec: func(ctx context.Context, eval subm.Eval, srcCode string, prLangId string) error {
			err := s.execSrvc.Enqueue(ctx, eval.UUID, srcCode, prLangId, nil, 0, 0, nil, nil)
			if err != nil {
				return fmt.Errorf("failed to enqueue evaluation: %w", err)
			}
			ch, err := s.execSrvc.Subscribe(ctx, eval.UUID)
			if err != nil {
				return fmt.Errorf("failed to subscribe to evaluation: %w", err)
			}
			go func(execEvCh <-chan execsrvc.Event) {
				for ev := range execEvCh {
					slog.Info("received execution event", "ev", ev)
				}
			}(ch)
			return nil
		},
	}

	return submitSolCmd.Handle(ctx, p)
}

func (s *SubmSrvc) ReEvalSubm(ctx context.Context, p submcmd.ReEvalSubmParams) error {
	panic("not implemented")
}

func (s *SubmSrvc) GetSubm(ctx context.Context, uuid uuid.UUID) (subm.Subm, error) {
	return s.submRepo.GetSubm(ctx, uuid)
}

func (s *SubmSrvc) ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]subm.Subm, error) {
	return s.submRepo.ListSubms(ctx, filter.Limit, filter.Offset)
}

func (s *SubmSrvc) GetEval(ctx context.Context, uuid uuid.UUID) (subm.Eval, error) {
	return s.evalRepo.GetEval(ctx, uuid)
}

func (s *SubmSrvc) SubsNewSubm(ctx context.Context) (<-chan subm.Subm, error) {
	ch := make(chan subm.Subm)
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

func (s *SubmSrvc) SubsEvalUpd(ctx context.Context) (<-chan subm.Eval, error) {
	ch := make(chan subm.Eval)
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
