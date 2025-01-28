package submquery

import (
	"context"

	decorator "github.com/programme-lv/backend/srvccqs"
	"github.com/programme-lv/backend/subm"
)

type SubsEvalUpd decorator.QueryHandler[SubsEvalUpdParams, <-chan subm.Eval]

type SubsEvalUpdParams struct{}

type subsEvalUpdHandler struct {
	subsAllEvalUpd func(ctx context.Context) (<-chan subm.Eval, error)
}

func NewSubsEvalUpdQuery(subsAllEvalUpd func(ctx context.Context) (<-chan subm.Eval, error)) SubsEvalUpd {
	return subsEvalUpdHandler{subsAllEvalUpd: subsAllEvalUpd}
}

func (h subsEvalUpdHandler) Handle(ctx context.Context, p SubsEvalUpdParams) (<-chan subm.Eval, error) {
	all, err := h.subsAllEvalUpd(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan subm.Eval, 1)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case eval := <-all:
				select {
				case <-ch: // drop old eval
				default:
				}
				ch <- eval // add new eval
			}
		}
	}()

	return ch, nil
}
