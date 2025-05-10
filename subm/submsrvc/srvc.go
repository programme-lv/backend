package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/exec"
	"github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
	"github.com/programme-lv/backend/task/srvc"
	usersrvc "github.com/programme-lv/backend/user"
	"github.com/programme-lv/backend/user/auth"
)

type SubmSrvcClient interface {
	SubmitSol(ctx context.Context, p submcmd.SubmitSolParams) error
	ReEvalSubm(ctx context.Context, submUuid uuid.UUID) error
	ViewSubm(ctx context.Context, uuid uuid.UUID) (domain.Subm, error)
	ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]domain.Subm, error)
	GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, error)
	SubscribeNewSubms(ctx context.Context) (<-chan domain.Subm, error)
	SubscribeEvalUpds(ctx context.Context) (<-chan domain.Eval, error)
	WaitForEvalFinish(ctx context.Context, evalUUID uuid.UUID) error
	GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, error)
	CountSubms(ctx context.Context) (int, error)
}

type submSrvc struct {
	submRepo SubmRepo
	evalRepo EvalRepo

	userSrvc UserSrvcFacade
	taskSrvc TaskSrvcFacade
	execSrvc ExecSrvcFacade

	newSubmChListenerLock sync.Mutex
	newSubmListeners      map[chan domain.Subm]struct{}

	newEvalUpdListenerLock sync.Mutex
	newEvalUpdListeners    map[chan domain.Eval]struct{}

	inProgrEval map[uuid.UUID]domain.Eval
}

type SubmRepo interface {
	AssignEval(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error
	GetSubm(ctx context.Context, id uuid.UUID) (domain.Subm, error)
	ListSubms(ctx context.Context, limit int, offset int) ([]domain.Subm, error)
	StoreSubm(ctx context.Context, subm domain.Subm) error
	CountSubms(ctx context.Context) (int, error)
}

type EvalRepo interface {
	GetEval(ctx context.Context, evalUUID uuid.UUID) (domain.Eval, error)
	StoreEval(ctx context.Context, eval domain.Eval) error
}

type UserSrvcFacade interface {
	GetUserByUUID(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error)
}

type TaskSrvcFacade interface {
	GetTask(ctx context.Context, shortId string) (srvc.Task, error)
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error)
}

type ExecSrvcFacade interface {
	Enqueue(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []exec.TestFile, params exec.TestingParams) error
	Listen(ctx context.Context, execUuid uuid.UUID) (<-chan exec.Event, error)
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
	userSrvc UserSrvcFacade,
	taskSrvc TaskSrvcFacade,
	execSrvc ExecSrvcFacade,
	submRepo SubmRepo,
	evalRepo EvalRepo,
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
	go func(execEvCh <-chan exec.Event) {
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
		EnqueueExec:      s.enqueueEvalExecAndListen,
	}

	return submitSolCmd.Handle(ctx, p)
}

func (s *submSrvc) ReEvalSubm(ctx context.Context, submUuid uuid.UUID) error {
	reevalSubmCmd := submcmd.ReEvalSubmHandler{
		GetSubm:     s.submRepo.GetSubm,
		GetTask:     s.taskSrvc.GetTask,
		StoreEval:   s.evalRepo.StoreEval,
		AssignEval:  s.submRepo.AssignEval,
		EnqueueExec: s.enqueueEvalExecAndListen,
	}
	return reevalSubmCmd.Handle(ctx, submUuid)
}

func (s *submSrvc) ViewSubm(ctx context.Context, submUuid uuid.UUID) (domain.Subm, error) {
	log := ctxlog.FromContext(ctx)

	subm, err := s.submRepo.GetSubm(ctx, submUuid)
	if err != nil {
		log.Error("failed to get submission", "error", err)
		return domain.Subm{}, fmt.Errorf("failed to get submission: %w", err)
	}

	userHasSolvedTheTask := false
	userUUID, err := auth.GetUserUuidFromCtx(ctx)
	if err == nil {
		userMaxScores, err := s.GetMaxScorePerTask(ctx, userUUID)
		if err != nil {
			log.Error("failed to get user scores", "error", err)
			return domain.Subm{}, fmt.Errorf("failed to get user scores: %w", err)
		}
		userScore, ok := userMaxScores[subm.TaskShortID]
		if !ok {
			log.Error("failed to get user score for task", "error", err)
			return domain.Subm{}, fmt.Errorf("failed to get user score for task: %w", err)
		}
		userHasSolvedTheTask = userScore.Received >= userScore.Possible
	}

	if !userHasSolvedTheTask {
		subm.Content = ""
	}

	return subm, nil
}

func (s *submSrvc) ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]domain.Subm, error) {
	log := ctxlog.FromContext(ctx)
	log.Debug("listing submissions", "limit", filter.Limit, "offset", filter.Offset)
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

// CountSubms returns the total number of submissions
func (s *submSrvc) CountSubms(ctx context.Context) (int, error) {
	log := ctxlog.FromContext(ctx)
	log.Debug("counting submissions")

	count, err := s.submRepo.CountSubms(ctx)
	if err != nil {
		log.Error("failed to count submissions", "error", err)
		return 0, err
	}

	return count, nil
}
