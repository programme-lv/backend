package submcmd

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	decorator "github.com/programme-lv/backend/srvccqs"
	subm "github.com/programme-lv/backend/subm/submdomain"
)

type ReEvalSubmCmd decorator.CmdHandler[ReEvalSubmParams]

func NewReEvalSubmCmd(
	getSubm func(ctx context.Context, submUuid uuid.UUID) (subm.Subm, error),
	createEval func(ctx context.Context, evalUuid uuid.UUID, submUuid uuid.UUID) error,
	assignEval func(ctx context.Context, evalUuid uuid.UUID) error,
	enqueueEval func(ctx context.Context, evalUuid uuid.UUID) error,
) ReEvalSubmCmd {
	return reEvalSubmHandler{
		getSubm:     getSubm,
		createEval:  createEval,
		assignEval:  assignEval,
		enqueueEval: enqueueEval,
	}
}

type ReEvalSubmParams struct {
	SubmUUID uuid.UUID
}

type reEvalSubmHandler struct {
	// get persisted submission entity by uuid
	getSubm func(ctx context.Context, submUuid uuid.UUID) (subm.Subm, error)

	// create and persist new evaluation entity, bcast evaluation created event
	createEval func(ctx context.Context, evalUuid uuid.UUID, submUuid uuid.UUID) error

	// assign evaluation to submission
	assignEval func(ctx context.Context, evalUuid uuid.UUID) error

	// enqueue evaluation for corresponding submission execution by tester
	enqueueEval func(ctx context.Context, evalUuid uuid.UUID) error
}

func (h reEvalSubmHandler) Handle(ctx context.Context, p ReEvalSubmParams) error {
	subm, err := h.getSubm(ctx, p.SubmUUID)
	if err != nil {
		return err
	}

	evalUuid := uuid.New()
	err = h.createEval(ctx, evalUuid, subm.UUID)
	if err != nil {
		return fmt.Errorf("failed to create evaluation: %w", err)
	}

	err = h.assignEval(ctx, evalUuid)
	if err != nil {
		return fmt.Errorf("failed to assign new eval to submission: %w", err)
	}

	err = h.enqueueEval(ctx, evalUuid)
	if err != nil {
		return fmt.Errorf("failed to enqueue evaluation: %w", err)
	}

	return nil
}
