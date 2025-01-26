package submcmds

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	decorator "github.com/programme-lv/backend/srvccqs"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submerrors"
)

type CreateSubmCmd decorator.CmdHandler[CreateSubmParams]

func NewCreateSubmCmd(
	persistSubm func(ctx context.Context, subm subm.Subm) error,
	doesUserExist func(ctx context.Context, uuid uuid.UUID) (bool, error),
	doesTaskExist func(ctx context.Context, shortId string) (bool, error),
	bcastNewSubmCreated func(subm subm.Subm),
) CreateSubmCmd {
	return createSubmHandler{
		storeSubm:           persistSubm,
		doesUserExist:       doesUserExist,
		doesTaskExist:       doesTaskExist,
		bcastNewSubmCreated: bcastNewSubmCreated,
	}
}

type createSubmHandler struct {
	storeSubm     func(ctx context.Context, subm subm.Subm) error
	doesUserExist func(ctx context.Context, uuid uuid.UUID) (bool, error)
	doesTaskExist func(ctx context.Context, shortId string) (bool, error)

	bcastNewSubmCreated func(subm subm.Subm)
}

type CreateSubmParams struct {
	UUID        uuid.UUID
	Submission  string
	AuthorUUID  uuid.UUID
	ProgrLangID string
	TaskShortID string
}

func (h createSubmHandler) Handle(ctx context.Context, p CreateSubmParams) error {
	if len(p.Submission) > 64*1024 { // 64 KB
		return submerrors.ErrSubmissionTooLong(64)
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
		AuthorUUID:   p.AuthorUUID,
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
