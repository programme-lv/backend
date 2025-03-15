package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subm struct {
	UUID         uuid.UUID
	Content      string
	AuthorUUID   uuid.UUID
	TaskShortID  string
	LangShortID  string
	CurrEvalUUID uuid.UUID
	CreatedAt    time.Time
}
