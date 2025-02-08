package submadapter

import (
	"context"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
	subm "github.com/programme-lv/backend/subm/submdomain"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"
)

type SubmRepo interface {
	AssignEval(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error
	GetSubm(ctx context.Context, id uuid.UUID) (subm.Subm, error)
	ListSubms(ctx context.Context, limit int, offset int) ([]subm.Subm, error)
	StoreSubm(ctx context.Context, subm subm.Subm) error
}

type EvalRepo interface {
	GetEval(ctx context.Context, evalUUID uuid.UUID) (subm.Eval, error)
	StoreEval(ctx context.Context, eval subm.Eval) error
}

type UserSrvcFacade interface {
	GetUserByUUID(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error)
}

type TaskSrvcFacade interface {
	GetTask(ctx context.Context, shortId string) (tasksrvc.Task, error)
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error)
}

type ExecSrvcFacade interface {
	Enqueue(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []execsrvc.TestFile, params execsrvc.TestingParams) error
	Listen(ctx context.Context, execUuid uuid.UUID) (<-chan execsrvc.Event, error)
}
