package submsrvc

import (
	"context"
	"fmt"
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
	AssignEvalCmd  submcmds.AssignEvalCmd
	EnqueueEvalCmd submcmds.EnqueueEvalCmd
	ReEvalSubmCmd  submcmds.ReEvalSubmCmd

	GetSubmQuery   submqueries.GetSubmQuery
	ListSubmsQuery submqueries.ListSubmsQuery
}

func NewSubmSrvc(
	userSrvc *usersrvc.UserService,
	taskSrvc *tasksrvc.TaskService,
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
	)

	createEvalCmd := submcmds.NewCreateEvalCmd(
		submRepo.GetSubm,
		taskSrvc.GetTask,
		evalRepo.StoreEval,
	)

	assignEvalCmd := submcmds.NewAssignEvalCmd(
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
		assignEvalFunc(assignEvalCmd),
		enqueueEvalFunc(enqueueEvalCmd),
	)

	getSubmQuery := submqueries.NewGetSubmQuery(submRepo.GetSubm)

	listSubmsQuery := submqueries.NewListSubmsQuery(submRepo.ListSubms)

	return &SubmSrvc{
		CreateSubmCmd:  createSubmCmd,
		CreateEvalCmd:  createEvalCmd,
		AssignEvalCmd:  assignEvalCmd,
		EnqueueEvalCmd: enqueueEvalCmd,
		ReEvalSubmCmd:  reevalSubmCmd,
		GetSubmQuery:   getSubmQuery,
		ListSubmsQuery: listSubmsQuery,
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

func assignEvalFunc(cmd submcmds.AssignEvalCmd) func(ctx context.Context, evalUuid uuid.UUID) error {
	return func(ctx context.Context, evalUuid uuid.UUID) error {
		return cmd.Handle(ctx, submcmds.AssignEvalParams{
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

func getUserUuidFunc(userSrvc *usersrvc.UserService) func(ctx context.Context, username string) (uuid.UUID, error) {
	return func(ctx context.Context, username string) (uuid.UUID, error) {
		user, err := userSrvc.GetUserByUsername(ctx, username)
		if err != nil {
			return uuid.Nil, err
		}
		return user.UUID, nil
	}
}

func getTaskExistsFunc(taskSrvc *tasksrvc.TaskService) func(ctx context.Context, shortId string) (bool, error) {
	return func(ctx context.Context, shortId string) (bool, error) {
		task, err := taskSrvc.GetTask(ctx, shortId)
		if err != nil {
			return false, err
		}
		return task.ShortId == shortId, nil
	}
}
