package submqueries

import (
	"context"

	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/decorator"
)

type ListSubmsQuery decorator.QueryHandler[ListSubmsParams, []subm.Subm]

func NewListSubmsQuery(listSubms func(ctx context.Context, limit int, offset int) ([]subm.Subm, error)) ListSubmsQuery {
	return listSubmsHandler{listSubms: listSubms}
}

type ListSubmsParams struct {
	Limit  int
	Offset int
}

type listSubmsHandler struct {
	listSubms func(ctx context.Context, limit int, offset int) ([]subm.Subm, error)
}

func (h listSubmsHandler) Handle(ctx context.Context, p ListSubmsParams) ([]subm.Subm, error) {
	subms, err := h.listSubms(ctx, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}

	return subms, nil
}
