package srvc

import (
	"context"
	"sync"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/exec"
	"github.com/programme-lv/backend/subm/domain"
	tasksrvc "github.com/programme-lv/backend/task/srvc"
	usersrvc "github.com/programme-lv/backend/user"
)

type SubmSrvcClient interface {
	SubmitSol(ctx context.Context, p SubmitSolParams) error
	ReEvalSubm(ctx context.Context, submUuid uuid.UUID) *srvcerror.Error
	ViewSubm(ctx context.Context, uuid uuid.UUID) (domain.Subm, error)
	ListSubms(ctx context.Context, filter ListSubmsParams) ([]domain.Subm, error)
	GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, error)
	SubscribeNewSubms(ctx context.Context) (<-chan domain.Subm, error)
	SubscribeEvalUpds(ctx context.Context) (<-chan domain.Eval, error)
	GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, *srvcerror.Error)
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

	// ListShallowSubmsJoinEval does not return submissions without a corresponding evaluation
	ListShallowSubmsJoinEval(ctx context.Context, authorUuid *uuid.UUID) ([]ShallowSubmJoinEvalDto, error)
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
