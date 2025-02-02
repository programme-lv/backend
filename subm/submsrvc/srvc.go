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

type submSrvc struct {
	userSrvc submadapter.UserSrvcFacade
	taskSrvc submadapter.TaskSrvcFacade
	execSrvc submadapter.ExecSrvcFacade
	submRepo submadapter.SubmRepo
	evalRepo submadapter.EvalRepo

	newSubmChListenerLock sync.Mutex
	newSubmListeners      map[chan<- subm.Subm]struct{}

	newEvalUpdListenerLock sync.Mutex
	newEvalUpdListeners    map[chan<- subm.Eval]struct{}

	inProgrEval map[uuid.UUID]subm.Eval
}

func NewSubmSrvc(
	userSrvc submadapter.UserSrvcFacade,
	taskSrvc submadapter.TaskSrvcFacade,
	execSrvc submadapter.ExecSrvcFacade,
	submRepo submadapter.SubmRepo,
	evalRepo submadapter.EvalRepo,
) SubmSrvcClient {
	return &submSrvc{
		userSrvc: userSrvc,
		taskSrvc: taskSrvc,
		execSrvc: execSrvc,
		submRepo: submRepo,
		evalRepo: evalRepo,

		newSubmListeners:    make(map[chan<- subm.Subm]struct{}),
		newEvalUpdListeners: make(map[chan<- subm.Eval]struct{}),

		inProgrEval: make(map[uuid.UUID]subm.Eval),
	}
}

func (s *submSrvc) procExecEv(ctx context.Context, p submcmd.ProcExecEvParams) error {
	procExecEvCmd := submcmd.ProcExecEvCmdHandler{
		StoreEval:     s.evalRepo.StoreEval,
		BcastEvalUpd:  s.broadcastEvalUpdate,
		GetEvalByUuid: s.evalRepo.GetEval,
		InProgrEval:   s.inProgrEval,
	}
	return procExecEvCmd.Handle(ctx, p)
}

func (s *submSrvc) broadcastEvalUpdate(eval subm.Eval) {
	s.newEvalUpdListenerLock.Lock()
	defer s.newEvalUpdListenerLock.Unlock()
	for ch := range s.newEvalUpdListeners {
		ch <- eval
	}
}

func (s *submSrvc) broadcastSubmCreated(subm subm.Subm) {
	s.newSubmChListenerLock.Lock()
	defer s.newSubmChListenerLock.Unlock()
	for ch := range s.newSubmListeners {
		ch <- subm
	}
}

func (s *submSrvc) enqueueEvalExecAndListen(ctx context.Context, eval subm.Eval, srcCode string, prLangId string) error {
	enqueueEvalCmd := submcmd.EnqueueEvalCmdHandler{
		EnqueueExec:     s.execSrvc.Enqueue,
		GetTestDownlUrl: s.taskSrvc.GetTestDownlUrl,
	}

	// Add eval to in-progress map before enqueueing
	s.inProgrEval[eval.UUID] = eval

	err := enqueueEvalCmd.Handle(ctx, submcmd.EnqueueEvalParams{
		Eval:     eval,
		SrcCode:  srcCode,
		PrLangId: prLangId,
	})
	if err != nil {
		delete(s.inProgrEval, eval.UUID) // Remove from map if enqueue fails
		return fmt.Errorf("failed to enqueue evaluation: %w", err)
	}

	ch, err := s.execSrvc.Listen(ctx, eval.UUID)
	if err != nil {
		delete(s.inProgrEval, eval.UUID) // Remove from map if listen fails
		return fmt.Errorf("failed to subscribe to evaluation: %w", err)
	}

	// Create a new background context for the event processing goroutine
	processCtx := context.Background()
	go func(execEvCh <-chan execsrvc.Event) {
		for ev := range execEvCh {
			err := s.procExecEv(processCtx, submcmd.ProcExecEvParams{
				Eval:  eval,
				Event: ev,
			})
			if err != nil {
				slog.Error("failed to process execution event", "error", err)
			}
		}
	}(ch)

	return nil
}

func (s *submSrvc) SubmitSol(ctx context.Context, p submcmd.SubmitSolParams) error {
	submitSolCmd := submcmd.SubmitSolCmdHandler{
		DoesUserExist: func(ctx context.Context, uuid uuid.UUID) (bool, error) {
			user, err := s.userSrvc.GetUserByUUID(ctx, uuid)
			if err != nil {
				return false, err
			}
			return user.UUID == uuid, nil
		},
		GetTask:          s.taskSrvc.GetTask,
		StoreSubm:        s.submRepo.StoreSubm,
		StoreEval:        s.evalRepo.StoreEval,
		BcastSubmCreated: s.broadcastSubmCreated,
		EnqueueEvalExec:  s.enqueueEvalExecAndListen,
	}

	return submitSolCmd.Handle(ctx, p)
}

func (s *submSrvc) ReEvalSubm(ctx context.Context, p submcmd.ReEvalSubmParams) error {
	panic("not implemented")
}

func (s *submSrvc) GetSubm(ctx context.Context, uuid uuid.UUID) (subm.Subm, error) {
	return s.submRepo.GetSubm(ctx, uuid)
}

func (s *submSrvc) ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]subm.Subm, error) {
	return s.submRepo.ListSubms(ctx, filter.Limit, filter.Offset)
}

func (s *submSrvc) GetEval(ctx context.Context, uuid uuid.UUID) (subm.Eval, error) {
	if eval, ok := s.inProgrEval[uuid]; ok {
		return eval, nil
	}
	return s.evalRepo.GetEval(ctx, uuid)
}

func (s *submSrvc) SubsNewSubm(ctx context.Context) (<-chan subm.Subm, error) {
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

func (s *submSrvc) SubsEvalUpd(ctx context.Context) (<-chan subm.Eval, error) {
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
