package submqueries

import (
	"context"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/decorator"
)

type GetEvalQuery decorator.QueryHandler[GetEvalParams, subm.Eval]

func NewGetEvalQuery(getEval func(ctx context.Context, evalUuid uuid.UUID) (subm.Eval, error)) GetEvalQuery {
	return getEvalHandler{getEval: getEval}
}

type GetEvalParams struct {
	EvalUUID uuid.UUID
}

type getEvalHandler struct {
	getEval func(ctx context.Context, evalUuid uuid.UUID) (subm.Eval, error)
}

func (h getEvalHandler) Handle(ctx context.Context, p GetEvalParams) (subm.Eval, error) {
	return h.getEval(ctx, p.EvalUUID)
}
