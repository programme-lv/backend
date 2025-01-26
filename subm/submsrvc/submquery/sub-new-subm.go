package submquery

import (
	"context"

	decorator "github.com/programme-lv/backend/srvccqs"
	"github.com/programme-lv/backend/subm"
)

type SubNewSubms decorator.QueryHandler[SubNewSubmsParams, <-chan subm.Subm]

type SubNewSubmsParams struct{}

func NewSubNewSubmsQuery(subNewSubms func(ctx context.Context) (<-chan subm.Subm, error)) SubNewSubms {
	return subNewSubmsHandler{subNewSubms: subNewSubms}
}

type subNewSubmsHandler struct {
	subNewSubms func(ctx context.Context) (<-chan subm.Subm, error)
}

func (h subNewSubmsHandler) Handle(ctx context.Context, p SubNewSubmsParams) (<-chan subm.Subm, error) {
	all, err := h.subNewSubms(ctx)
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
