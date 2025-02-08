package submquery

import (
	"context"

	"github.com/google/uuid"
	decorator "github.com/programme-lv/backend/srvccqs"
	subm "github.com/programme-lv/backend/subm/submdomain"
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
