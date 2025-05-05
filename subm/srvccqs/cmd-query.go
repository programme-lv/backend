package decorator

import "context"

// P - params
type CmdHandler[P any] interface {
	Handle(ctx context.Context, p P) error
}

// Q - query, R - result
type QueryHandler[Q any, R any] interface {
	Handle(ctx context.Context, q Q) (R, error)
}
