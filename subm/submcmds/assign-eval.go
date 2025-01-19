package submcmds

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/decorator"
)

type AssignEvalCmd decorator.CmdHandler[AssignEvalParams]

func NewAssignEvalCmd(
	getEval func(ctx context.Context, uuid uuid.UUID) (subm.Eval, error),
	getSubm func(ctx context.Context, uuid uuid.UUID) (subm.Subm, error),
	storeSubm func(ctx context.Context, subm subm.Subm) error,
) AssignEvalCmd {
	return assignEvalHandler{
		getEval:   getEval,
		getSubm:   getSubm,
		storeSubm: storeSubm,
	}
}

type AssignEvalParams struct {
	EvalUUID uuid.UUID
}

type assignEvalHandler struct {
	getEval   func(ctx context.Context, uuid uuid.UUID) (subm.Eval, error)
	getSubm   func(ctx context.Context, uuid uuid.UUID) (subm.Subm, error)
	storeSubm func(ctx context.Context, subm subm.Subm) error
}

func (h assignEvalHandler) Handle(ctx context.Context, p AssignEvalParams) error {
	eval, err := h.getEval(ctx, p.EvalUUID)
	if err != nil {
		return fmt.Errorf("failed to get eval: %w", err)
	}

	subm, err := h.getSubm(ctx, eval.SubmUUID)
	if err != nil {
		return fmt.Errorf("failed to get subm: %w", err)
	}

	subm.CurrEvalUUID = eval.UUID

	if err := h.storeSubm(ctx, subm); err != nil {
		return fmt.Errorf("failed to store subm: %w", err)
	}

	return nil
}
