package submquery

import (
	"context"

	subm "github.com/programme-lv/backend/subm/domain"
	decorator "github.com/programme-lv/backend/subm/srvccqs"
)

type ListSubmsQuery decorator.QueryHandler[ListSubmsParams, []subm.Subm]

func NewListSubmsQuery(listSubms func(ctx context.Context, limit int, offset int, search string) ([]subm.Subm, error)) ListSubmsQuery {
	return listSubmsHandler{listSubms: listSubms}
}

type ListSubmsParams struct {
	Limit  int
	Offset int
	Search string
}

type listSubmsHandler struct {
	listSubms func(ctx context.Context, limit int, offset int, search string) ([]subm.Subm, error)
}

func (h listSubmsHandler) Handle(ctx context.Context, p ListSubmsParams) ([]subm.Subm, error) {
	subms, err := h.listSubms(ctx, p.Limit, p.Offset, p.Search)
	if err != nil {
		return nil, err
	}

	return subms, nil
}
