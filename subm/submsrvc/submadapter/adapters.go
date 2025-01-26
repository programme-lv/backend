package submadapter

import (
	"context"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm"
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
}

type ExecSrvcFacade interface {
	Enqueue(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []execsrvc.TestFile, cpuMs int, memKiB int, checker *string, interactor *string) error
	Subscribe(ctx context.Context, evalUuid uuid.UUID) (<-chan execsrvc.Event, error)
}
