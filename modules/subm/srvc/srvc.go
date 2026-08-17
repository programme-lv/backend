package srvc

import (
	"context"
	"sync"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/exec"
	"github.com/programme-lv/backend/modules/subm/domain"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
	usersrvc "github.com/programme-lv/backend/modules/user"
)

// SubmissionService is the port (service contract) for it's bounded context
type SubmissionService interface {
	SubmitSol(ctx context.Context, p SubmitSolParams) srvcerror.E
	ReEvalSubm(ctx context.Context, submUuid uuid.UUID) srvcerror.E
	ViewSubm(ctx context.Context, uuid uuid.UUID) (domain.Subm, srvcerror.E)
	ViewSubmByShortID(ctx context.Context, shortID string) (domain.Subm, srvcerror.E)
	ListSubms(ctx context.Context, filter ListSubmsParams) ([]domain.Subm, srvcerror.E)
	GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, srvcerror.E)
	SubscribeNewSubms(ctx context.Context) (<-chan domain.Subm, srvcerror.E)
	SubscribeEvalUpds(ctx context.Context) (<-chan domain.Eval, srvcerror.E)
	GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, srvcerror.E)
	CountSubms(ctx context.Context, search string, author *uuid.UUID, includeAdmin bool) (int, srvcerror.E)
}

var _ SubmissionService = &submSrvc{}

type submSrvc struct {
	submRepo SubmRepo
	evalRepo EvalRepo

	userSrvc usersrvc.UserService
	taskSrvc tasksrvc.TaskService
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
	GetSubmByShortID(ctx context.Context, shortID string) (domain.Subm, error)
	ListSubms(ctx context.Context, limit int, offset int, search string, authorUuid *uuid.UUID, authorIds []string, taskIds []string, langIds []string, includeAdmin bool) ([]domain.Subm, error)
	// ListSubmsJoinEval(ctx context.Context, authorUuid *uuid.UUID) ([]domain.SubmJoinEval, error)
	StoreSubm(ctx context.Context, subm *domain.Subm) error
	CountSubms(ctx context.Context, authorUuid *uuid.UUID, authorIds []string, taskIds []string, langIds []string, includeAdmin bool) (int, error)

	// ListShallowSubmsJoinEval does not return submissions without a corresponding evaluation
	ListShallowSubmsJoinEval(ctx context.Context, authorUuid *uuid.UUID) ([]ShallowSubmJoinEvalDto, error)
}

type EvalRepo interface {
	GetEval(ctx context.Context, evalUUID uuid.UUID) (domain.Eval, error)
	StoreEval(ctx context.Context, eval domain.Eval) error
}

type ExecSrvcFacade interface {
	Enqueue(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []exec.TestFile, params exec.TestingParams) srvcerror.E
	Listen(ctx context.Context, execUuid uuid.UUID) (<-chan exec.Event, srvcerror.E)
}

func NewSubmSrvc(
	userSrvc usersrvc.UserService,
	taskSrvc tasksrvc.TaskService,
	execSrvc ExecSrvcFacade,
	submRepo SubmRepo,
	evalRepo EvalRepo,
) *submSrvc {
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
