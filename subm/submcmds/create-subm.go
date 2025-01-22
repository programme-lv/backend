package submcmds

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/decorator"
	"github.com/programme-lv/backend/subm/submerrors"
)

type CreateSubmCmd decorator.CmdHandler[CreateSubmParams]

func NewCreateSubmCmd(
	getUserUuid func(ctx context.Context, username string) (uuid.UUID, error),
	persistSubm func(ctx context.Context, subm subm.Subm) error,
	doesTaskExist func(ctx context.Context, shortId string) (bool, error),
	bcastNewSubmCreated func(subm subm.Subm),
) CreateSubmCmd {
	return createSubmHandler{
		getUserUuid:         getUserUuid,
		storeSubm:           persistSubm,
		doesTaskExist:       doesTaskExist,
		bcastNewSubmCreated: bcastNewSubmCreated,
	}
}

type createSubmHandler struct {
	getUserUuid   func(ctx context.Context, username string) (uuid.UUID, error)
	storeSubm     func(ctx context.Context, subm subm.Subm) error
	doesTaskExist func(ctx context.Context, shortId string) (bool, error)

	bcastNewSubmCreated func(subm subm.Subm)
}

type CreateSubmParams struct {
	UUID        uuid.UUID
	Submission  string
	Username    string
	ProgrLangID string
	TaskShortID string
}

func (h createSubmHandler) Handle(ctx context.Context, p CreateSubmParams) error {
	if len(p.Submission) > 64*1024 { // 64 KB
		return submerrors.ErrSubmissionTooLong(64)
	}

	uUuid, err := h.getUserUuid(ctx, p.Username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	l, err := planglist.GetProgrLangById(p.ProgrLangID)
	if err != nil {
		return fmt.Errorf("failed to get progr lang: %w", err)
	}

	taskExists, err := h.doesTaskExist(ctx, p.TaskShortID)
	if err != nil {
		return fmt.Errorf("failed to check if task exists: %w", err)
	}
	if !taskExists {
		return submerrors.ErrTaskNotFound()
	}

	entity := subm.Subm{
		UUID:         p.UUID,
		Content:      p.Submission,
		AuthorUUID:   uUuid,
		TaskShortID:  p.TaskShortID,
		LangShortID:  l.ID,
		CurrEvalUUID: uuid.Nil,
		CreatedAt:    time.Now(),
	}

	err = h.storeSubm(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to store submission: %w", err)
	}

	h.bcastNewSubmCreated(entity)

	return nil
}
