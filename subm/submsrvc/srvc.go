package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submcmds"
	"github.com/programme-lv/backend/subm/submpgrepo"
	"github.com/programme-lv/backend/subm/submqueries"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"
)

type SubmSrvc struct {
	CreateSubmCmd  submcmds.CreateSubmCmd
	CreateEvalCmd  submcmds.CreateEvalCmd
	AttachEvalCmd  submcmds.AttachEvalCmd
	EnqueueEvalCmd submcmds.EnqueueEvalCmd
	ReEvalSubmCmd  submcmds.ReEvalSubmCmd

	GetSubmQuery   submqueries.GetSubmQuery
	ListSubmsQuery submqueries.ListSubmsQuery
	GetEvalQuery   submqueries.GetEvalQuery
}

func NewSubmSrvc(
	userSrvc *usersrvc.UserSrvc,
	taskSrvc *tasksrvc.TaskSrvc,
	execSrvc *execsrvc.ExecSrvc,
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
		getUserUuidFunc(userSrvc),
		submRepo.StoreSubm,
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
		CreateSubmCmd:  createSubmCmd,
		CreateEvalCmd:  createEvalCmd,
		AttachEvalCmd:  attachEvalCmd,
		EnqueueEvalCmd: enqueueEvalCmd,
		ReEvalSubmCmd:  reevalSubmCmd,
		GetSubmQuery:   getSubmQuery,
		ListSubmsQuery: listSubmsQuery,
		GetEvalQuery:   getEvalQuery,
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

func getUserUuidFunc(userSrvc *usersrvc.UserSrvc) func(ctx context.Context, username string) (uuid.UUID, error) {
	return func(ctx context.Context, username string) (uuid.UUID, error) {
		user, err := userSrvc.GetUserByUsername(ctx, username)
		if err != nil {
			return uuid.Nil, err
		}
		return user.UUID, nil
	}
}

func getTaskExistsFunc(taskSrvc *tasksrvc.TaskSrvc) func(ctx context.Context, shortId string) (bool, error) {
	return func(ctx context.Context, shortId string) (bool, error) {
		task, err := taskSrvc.GetTask(ctx, shortId)
		if err != nil {
			return false, err
		}
		return task.ShortId == shortId, nil
	}
}
