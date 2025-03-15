package submquery

import (
	"context"

	decorator "github.com/programme-lv/backend/srvccqs"
	subm "github.com/programme-lv/backend/subm/domain"
)

type SubsNewSubms decorator.QueryHandler[SubsNewSubmsParams, <-chan subm.Subm]

type SubsNewSubmsParams struct{}

func NewSubsNewSubmsQuery(subNewSubms func(ctx context.Context) (<-chan subm.Subm, error)) SubsNewSubms {
	return subsNewSubmsHandler{subsNewSubms: subNewSubms}
}

type subsNewSubmsHandler struct {
	subsNewSubms func(ctx context.Context) (<-chan subm.Subm, error)
}

func (h subsNewSubmsHandler) Handle(ctx context.Context, p SubsNewSubmsParams) (<-chan subm.Subm, error) {
	all, err := h.subsNewSubms(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan subm.Subm, 10)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case subm := <-all:
				if len(ch) == cap(ch) {
					<-ch
				}
				ch <- subm
			}
		}
	}()
	return ch, nil
}
