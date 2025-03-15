package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/submsrvc/submadapter"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
)

type SubmSrvcClient interface {
	SubmitSol(ctx context.Context, p submcmd.SubmitSolParams) error
	ReEvalSubm(ctx context.Context, p submcmd.ReEvalSubmParams) error
	GetSubm(ctx context.Context, uuid uuid.UUID) (domain.Subm, error)
	ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]domain.Subm, error)
	GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, error)
	SubscribeNewSubms(ctx context.Context) (<-chan domain.Subm, error)
	SubscribeEvalUpds(ctx context.Context) (<-chan domain.Eval, error)
	WaitForEvalFinish(ctx context.Context, evalUUID uuid.UUID) error
	GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, error)
}

type submSrvc struct {
	userSrvc submadapter.UserSrvcFacade
	taskSrvc submadapter.TaskSrvcFacade
	execSrvc submadapter.ExecSrvcFacade
	submRepo submadapter.SubmRepo
	evalRepo submadapter.EvalRepo

	newSubmChListenerLock sync.Mutex
	newSubmListeners      map[chan domain.Subm]struct{}

	newEvalUpdListenerLock sync.Mutex
	newEvalUpdListeners    map[chan domain.Eval]struct{}

	inProgrEval map[uuid.UUID]domain.Eval
}

// GetMaxScorePerTask implements SubmSrvcClient.
func (s *submSrvc) GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, error) {
	// Get all submissions
	subms, err := s.submRepo.ListSubms(ctx, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}

	if len(subms) == 10000 {
		slog.Error("too many submissions", "user_uuid", userUUID)
	}

	// Filter submissions by user and collect evaluations
	userSubmsWithEval := make([]domain.SubmJoinEval, 0)
	for _, subm := range subms {
		if subm.AuthorUUID != userUUID {
			continue
		}

		// Skip submissions without evaluations
		if subm.CurrEvalUUID == uuid.Nil {
			continue
		}

		// Get the evaluation
		eval, err := s.GetEval(ctx, subm.CurrEvalUUID)
		if err != nil {
			slog.Error("failed to get evaluation", "error", err, "eval_uuid", subm.CurrEvalUUID)
			continue
		}

		userSubmsWithEval = append(userSubmsWithEval, domain.SubmJoinEval{
			Subm: subm,
			Eval: eval,
		})
	}

	// Calculate max scores using the domain logic
	return domain.CalcMaxScores(userSubmsWithEval), nil
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

		newSubmListeners:    make(map[chan domain.Subm]struct{}),
		newEvalUpdListeners: make(map[chan domain.Eval]struct{}),

		inProgrEval: make(map[uuid.UUID]domain.Eval),
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

func (s *submSrvc) enqueueEvalExecAndListen(ctx context.Context, eval domain.Eval, srcCode string, prLangId string) error {
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

func (s *submSrvc) GetSubm(ctx context.Context, uuid uuid.UUID) (domain.Subm, error) {
	return s.submRepo.GetSubm(ctx, uuid)
}

func (s *submSrvc) ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]domain.Subm, error) {
	return s.submRepo.ListSubms(ctx, filter.Limit, filter.Offset)
}

func (s *submSrvc) GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, error) {
	if eval, ok := s.inProgrEval[uuid]; ok {
		return eval, nil
	}
	return s.evalRepo.GetEval(ctx, uuid)
}

func (s *submSrvc) SubscribeNewSubms(ctx context.Context) (<-chan domain.Subm, error) {
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

func (s *submSrvc) SubscribeEvalUpds(ctx context.Context) (<-chan domain.Eval, error) {
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

func (s *submSrvc) WaitForEvalFinish(ctx context.Context, evalUUID uuid.UUID) error {
	subscrCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	evalUpdCh, err := s.SubscribeEvalUpds(subscrCtx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to evaluation updates: %w", err)
	}

	eval, err := s.GetEval(ctx, evalUUID)
	if err != nil {
		return fmt.Errorf("failed to get evaluation: %w", err)
	}

	if eval.Stage == domain.EvalStageFinished {
		return nil
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case e, ok := <-evalUpdCh:
			if !ok {
				return fmt.Errorf("failed to subscribe to evaluation updates")
			}
			if e.UUID != evalUUID {
				continue
			}
			if e.Stage == domain.EvalStageFinished {
				return nil
			}
			// extend timeout
			timeout = time.After(5 * time.Second)
		case <-timeout:
			return fmt.Errorf("timed out waiting for evaluation updates")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
