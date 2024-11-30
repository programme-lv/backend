package submsrvc

import (
	"context"

	"github.com/google/uuid"
)

type submRow struct {
	Uuid       uuid.UUID
	Content    string
	AuthorUUID uuid.UUID
	TaskID     string
	LangID     string
}

type SubmRepo interface {
	Store(ctx context.Context, subm submRow) error
}
