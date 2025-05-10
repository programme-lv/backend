package submcmd

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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
	GetTask func(ctx context.Context, shortId string) (tasksrvc.Task, error)

	// persist evaluation entity
	StoreEval func(ctx context.Context, eval submdomain.Eval) error

	// assign evaluation to submission
	AssignEval func(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error

	// enqueue evaluation for corresponding submission execution by tester
	EnqueueExec func(ctx context.Context, eval submdomain.Eval, srcCode string, prLangId string) error
}

func (h ReEvalSubmHandler) Handle(ctx context.Context, submUuid uuid.UUID) error {
	subm, err := h.GetSubm(ctx, submUuid)
	if err != nil {
		return err
	}

	t, err := h.GetTask(ctx, subm.TaskShortID)
	if err != nil {
		errMsg := fmt.Errorf("failed to get task: %w", err)
		return errMsg
	}

	evalUuid := uuid.New()
	eval := submdomain.NewEval(evalUuid, subm.UUID, t)

	err = h.StoreEval(ctx, eval)
	if err != nil {
		return fmt.Errorf("failed to store evaluation: %w", err)
	}

	err = h.AssignEval(ctx, subm.UUID, evalUuid)
	if err != nil {
		return fmt.Errorf("failed to assign new eval to submission: %w", err)
	}

	err = h.EnqueueExec(ctx, eval, subm.Content, subm.LangShortID)
	if err != nil {
		return fmt.Errorf("failed to enqueue evaluation: %w", err)
	}

	return nil
}
