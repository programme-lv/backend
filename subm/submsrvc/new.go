package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submpgrepo"
	"github.com/programme-lv/backend/subm/submsrvc/submcmds"
	"github.com/programme-lv/backend/subm/submsrvc/submqueries"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"
)

type UserSrvcFacade interface {
	GetUserByUUID(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error)
}

type TaskSrvcFacade interface {
	GetTask(ctx context.Context, shortId string) (tasksrvc.Task, error)
}

type ExecSrvcFacade interface {
	Enqueue(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []execsrvc.TestFile, params execsrvc.TesterParams) (uuid.UUID, error)
	Listen(ctx context.Context, evalUuid uuid.UUID) (<-chan execsrvc.Event, error)
}

func NewSubmSrvc(
	userSrvc UserSrvcFacade,
	taskSrvc TaskSrvcFacade,
	execSrvc ExecSrvcFacade,
) (*SubmSrvc, error) {
	pgPool, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to pg: %w", err)
	}

	submRepo := submpgrepo.NewPgSubmRepo(pgPool)
	evalRepo := submpgrepo.NewPgEvalRepo(pgPool)

	lock := sync.Mutex{}
	evalMem := map[uuid.UUID]subm.Eval{}
	updateEvalInMem := func(eval subm.Eval) {
		lock.Lock()
		defer lock.Unlock()
		evalMem[eval.UUID] = eval
	}

	updateEvalFinal := func(ctx context.Context, eval subm.Eval) error {
		err := evalRepo.StoreEval(ctx, eval)
		if err != nil {
			return fmt.Errorf("failed to store eval: %w", err)
		}
		lock.Lock()
		defer lock.Unlock()
		delete(evalMem, eval.UUID)
		return nil
	}

	createSubmCmd := submcmds.NewCreateSubmCmd(
		submRepo.StoreSubm,
		getUserExistsFunc(userSrvc),
		getTaskExistsFunc(taskSrvc),
		func(subm subm.Subm) {
			slog.Default().Info("NewSubmCreated", "subm", subm)
		},
	)

	createEvalCmd := submcmds.NewCreateEvalCmd(
		submRepo.GetSubm,
		taskSrvc.GetTask,
		evalRepo.StoreEval,
	)

	attachEvalCmd := submcmds.NewAttachEvalCmd(
		evalRepo.GetEval,
		submRepo.GetSubm,
		submRepo.StoreSubm,
	)

	enqueueEvalCmd := submcmds.NewEnqueueEvalCmd(
		submRepo.GetSubm,
		evalRepo.GetEval,
		execSrvc.Enqueue,
		execSrvc.Listen,
		updateEvalInMem,
		updateEvalFinal,
	)

	reevalSubmCmd := submcmds.NewReEvalSubmCmd(
		submRepo.GetSubm,
		createEvalFunc(createEvalCmd),
		attachEvalFunc(attachEvalCmd),
		enqueueEvalFunc(enqueueEvalCmd),
	)

	getSubmQuery := submqueries.NewGetSubmQuery(submRepo.GetSubm)
	listSubmsQuery := submqueries.NewListSubmsQuery(submRepo.ListSubms)
	getEvalQuery := submqueries.NewGetEvalQuery(evalRepo.GetEval)

	return &SubmSrvc{
		CreateSubm:  createSubmCmd,
		CreateEval:  createEvalCmd,
		AttachEval:  attachEvalCmd,
		EnqueueEval: enqueueEvalCmd,
		ReEvalSubm:  reevalSubmCmd,
		GetSubm:     getSubmQuery,
		ListSubms:   listSubmsQuery,
		GetEval:     getEvalQuery,
	}, nil
}

func createEvalFunc(cmd submcmds.CreateEvalCmd) func(ctx context.Context, evalUuid uuid.UUID, submUuid uuid.UUID) error {
	return func(ctx context.Context, evalUuid uuid.UUID, submUuid uuid.UUID) error {
		return cmd.Handle(ctx, submcmds.CreateEvalParams{
			EvalUUID: evalUuid,
			SubmUUID: submUuid,
		})
	}
}

func attachEvalFunc(cmd submcmds.AttachEvalCmd) func(ctx context.Context, evalUuid uuid.UUID) error {
	return func(ctx context.Context, evalUuid uuid.UUID) error {
		return cmd.Handle(ctx, submcmds.AttachEvalParams{
			EvalUUID: evalUuid,
		})
	}
}

func enqueueEvalFunc(cmd submcmds.EnqueueEvalCmd) func(ctx context.Context, evalUuid uuid.UUID) error {
	return func(ctx context.Context, evalUuid uuid.UUID) error {
		return cmd.Handle(ctx, submcmds.EnqueueEvalParams{
			EvalUUID: evalUuid,
		})
	}
}

func getUserExistsFunc(userSrvc UserSrvcFacade) func(ctx context.Context, uuid uuid.UUID) (bool, error) {
	return func(ctx context.Context, uuid uuid.UUID) (bool, error) {
		u, err := userSrvc.GetUserByUUID(ctx, uuid)
		if err != nil {
			return false, err
		}
		return u.UUID == uuid, nil
	}
}

func getTaskExistsFunc(taskSrvc TaskSrvcFacade) func(ctx context.Context, shortId string) (bool, error) {
	return func(ctx context.Context, shortId string) (bool, error) {
		t, err := taskSrvc.GetTask(ctx, shortId)
		if err != nil {
			return false, err
		}
		return t.ShortId == shortId, nil
	}
}
