package submcmd

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/srvcerror"
	submdomain "github.com/programme-lv/backend/subm/domain"
	tasksrvc "github.com/programme-lv/backend/task/srvc"
)

type ReEvalSubmParams struct {
	SubmUUID uuid.UUID
}

type ReEvalSubmHandler struct {
	// get persisted submission entity by uuid
	GetSubm func(ctx context.Context, submUuid uuid.UUID) (submdomain.Subm, error)

	// get persisted task entity by short id
	GetTask func(ctx context.Context, shortId string) (tasksrvc.Task, *srvcerror.Error)

	// persist evaluation entity
	StoreEval func(ctx context.Context, eval submdomain.Eval) error

	// assign evaluation to submission
	AssignEval func(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error

	// enqueue evaluation for corresponding submission execution by tester
	EnqueueExec func(ctx context.Context, eval submdomain.Eval, srcCode string, prLangId string) error
}

func (h ReEvalSubmHandler) Handle(ctx context.Context, submUuid uuid.UUID) error {
	subm, getSubmErr := h.GetSubm(ctx, submUuid)
	if getSubmErr != nil {
		return getSubmErr
	}

	t, getTaskErr := h.GetTask(ctx, subm.TaskShortID)
	if getTaskErr != nil {
		errMsg := fmt.Errorf("failed to get task: %w", getTaskErr)
		return errMsg
	}

	evalUuid := uuid.New()
	eval := submdomain.NewEval(evalUuid, subm.UUID, t)

	storeEvalErr := h.StoreEval(ctx, eval)
	if storeEvalErr != nil {
		return fmt.Errorf("failed to store evaluation: %w", storeEvalErr)
	}

	assignEvalErr := h.AssignEval(ctx, subm.UUID, evalUuid)
	if assignEvalErr != nil {
		return fmt.Errorf("failed to assign new eval to submission: %w", assignEvalErr)
	}

	enqueueExecErr := h.EnqueueExec(ctx, eval, subm.Content, subm.LangShortID)
	if enqueueExecErr != nil {
		return fmt.Errorf("failed to enqueue evaluation: %w", enqueueExecErr)
	}

	return nil
}
