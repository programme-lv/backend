package srvc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/exec"
	"github.com/programme-lv/backend/plang"
	"github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/srvc/submquery"
	tasksrvc "github.com/programme-lv/backend/task/srvc"
	usersrvc "github.com/programme-lv/backend/user"
	"github.com/programme-lv/backend/user/auth"
)

type SubmSrvcClient interface {
	SubmitSol(ctx context.Context, p SubmitSolParams) error
	ReEvalSubm(ctx context.Context, submUuid uuid.UUID) *srvcerror.Error
	ViewSubm(ctx context.Context, uuid uuid.UUID) (domain.Subm, error)
	ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]domain.Subm, error)
	GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, error)
	SubscribeNewSubms(ctx context.Context) (<-chan domain.Subm, error)
	SubscribeEvalUpds(ctx context.Context) (<-chan domain.Eval, error)
	WaitForEvalFinish(ctx context.Context, evalUUID uuid.UUID) error
	GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, error)
	CountSubms(ctx context.Context, search string, author *uuid.UUID) (int, error)
}

type submSrvc struct {
	submRepo SubmRepo
	evalRepo EvalRepo

	userSrvc usersrvc.UserSrvcClient
	taskSrvc tasksrvc.TaskSrvcClient
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
	ListSubms(ctx context.Context, limit int, offset int, search string, authorUuid *uuid.UUID, authorIds []string, taskIds []string, langIds []string) ([]domain.Subm, error)
	// ListSubmsJoinEval(ctx context.Context, authorUuid *uuid.UUID) ([]domain.SubmJoinEval, error)
	StoreSubm(ctx context.Context, subm domain.Subm) error
	CountSubms(ctx context.Context, authorUuid *uuid.UUID, authorIds []string, taskIds []string, langIds []string) (int, error)
}

type EvalRepo interface {
	GetEval(ctx context.Context, evalUUID uuid.UUID) (domain.Eval, error)
	StoreEval(ctx context.Context, eval domain.Eval) error
}

type ExecSrvcFacade interface {
	Enqueue(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []exec.TestFile, params exec.TestingParams) error
	Listen(ctx context.Context, execUuid uuid.UUID) (<-chan exec.Event, error)
}

func NewSubmSrvc(
	userSrvc usersrvc.UserSrvcClient,
	taskSrvc tasksrvc.TaskSrvcClient,
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

func (s *submSrvc) procExecEv(ctx context.Context, p ProcExecEvParams) error {
	procExecEvCmd := ProcExecEvCmdHandler{
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

func (s *submSrvc) ViewSubm(ctx context.Context, submUuid uuid.UUID) (domain.Subm, error) {
	log := ctxlog.FromContext(ctx)

	subm, err := s.submRepo.GetSubm(ctx, submUuid)
	if err != nil {
		log.Error("failed to get submission", "error", err)
		return domain.Subm{}, fmt.Errorf("failed to get submission: %w", err)
	}

	userHasSolvedTheTask := false
	userUUID, err := auth.GetUserUuidFromCtx(ctx)
	if err == nil && userUUID != subm.AuthorUUID {
		userMaxScores, err := s.GetMaxScorePerTask(ctx, userUUID)
		if err != nil {
			log.Error("failed to get user scores", "error", err)
			return domain.Subm{}, fmt.Errorf("failed to get user scores: %w", err)
		}
		userScore, ok := userMaxScores[subm.TaskShortID]
		if ok {
			userHasSolvedTheTask = userScore.Received >= userScore.Possible
		}
	}

	if !userHasSolvedTheTask && subm.AuthorUUID != userUUID {
		subm.Content = ""
	}

	return subm, nil
}

func (s *submSrvc) ListSubms(ctx context.Context, filter submquery.ListSubmsParams) ([]domain.Subm, error) {
	log := ctxlog.FromContext(ctx)
	log.Debug("listing submissions", "limit", filter.Limit, "offset", filter.Offset)

	authorIds := make([]string, 0)
	taskIds := make([]string, 0)
	langIds := make([]string, 0)

	if filter.Search != "" {
		// get all possible task matches
		var searchTasksByNameErr *srvcerror.Error
		taskIds, searchTasksByNameErr = s.taskSrvc.SearchTasksByName(ctx, filter.Search)
		if searchTasksByNameErr != nil {
			return nil, fmt.Errorf("failed to search tasks by name: %w", searchTasksByNameErr)
		}
		taskIds = append(taskIds, filter.Search)

		// get all possible user matches
		authorId, err := s.userSrvc.GetUserByUsername(ctx, filter.Search)
		if err != nil && !srvcerror.Is(err, usersrvc.ErrCodeUserNotFound) {
			return nil, fmt.Errorf("failed to get user by username: %w", err)
		}
		if authorId.UUID != uuid.Nil {
			authorIdStr := authorId.UUID.String()
			authorIds = append(authorIds, authorIdStr)
		}
		if _, err := uuid.Parse(filter.Search); err == nil {
			authorIds = append(authorIds, filter.Search)
		}

		// get all programming languages
		langIds, err = plang.SearchProgrLangByName(filter.Search)
		if err != nil {
			return nil, fmt.Errorf("failed to get programming language by id: %w", err)
		}
		langIds = append(langIds, filter.Search)
	}
	return s.submRepo.ListSubms(ctx, filter.Limit, filter.Offset, filter.Search, filter.Author, authorIds, taskIds, langIds)
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
func (s *submSrvc) CountSubms(ctx context.Context, search string, author *uuid.UUID) (int, error) {
	log := ctxlog.FromContext(ctx)
	log.Debug("counting submissions")

	authorIds := make([]string, 0)
	taskIds := make([]string, 0)
	langIds := make([]string, 0)

	if search != "" {
		// get all possible task matches
		var searchTasksByNameErr *srvcerror.Error
		taskIds, searchTasksByNameErr = s.taskSrvc.SearchTasksByName(ctx, search)
		if searchTasksByNameErr != nil {
			return 0, fmt.Errorf("failed to search tasks by name: %w", searchTasksByNameErr)
		}
		taskIds = append(taskIds, search)

		// get all possible user matches
		authorId, err := s.userSrvc.GetUserByUsername(ctx, search)
		if err != nil && !srvcerror.Is(err, usersrvc.ErrCodeUserNotFound) {
			return 0, fmt.Errorf("failed to get user by username: %w", err)
		}
		if authorId.UUID != uuid.Nil {
			authorIdStr := authorId.UUID.String()
			authorIds = append(authorIds, authorIdStr)
		}
		if _, err := uuid.Parse(search); err == nil {
			authorIds = append(authorIds, search)
		}

		// get all programming languages
		langIds, err = plang.SearchProgrLangByName(search)
		if err != nil {
			return 0, fmt.Errorf("failed to get programming language by id: %w", err)
		}
		langIds = append(langIds, search)
	}
	count, err := s.submRepo.CountSubms(ctx, author, authorIds, taskIds, langIds)
	if err != nil {
		log.Error("failed to count submissions", "error", err)
		return 0, err
	}

	return count, nil
}
