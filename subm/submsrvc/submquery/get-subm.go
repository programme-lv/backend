package submquery

import (
	"context"

	"github.com/google/uuid"
	subm "github.com/programme-lv/backend/subm/domain"
	decorator "github.com/programme-lv/backend/subm/srvccqs"
)

type GetSubmQuery decorator.QueryHandler[GetSubmParams, subm.Subm]

func NewGetSubmQuery(getSubm func(ctx context.Context, submUuid uuid.UUID) (subm.Subm, error)) GetSubmQuery {
	return getSubmHandler{getSubm: getSubm}
}

type GetSubmParams struct {
	SubmUUID uuid.UUID
}

type getSubmHandler struct {
	getSubm func(ctx context.Context, submUuid uuid.UUID) (subm.Subm, error)
}

func (s getSubmHandler) Handle(ctx context.Context, p GetSubmParams) (subm.Subm, error) {
	return s.getSubm(ctx, p.SubmUUID)
}
