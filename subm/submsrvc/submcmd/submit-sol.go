package submcmd

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	decorator "github.com/programme-lv/backend/srvccqs"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submerror"
	"github.com/programme-lv/backend/tasksrvc"
)

type SubmitSolCmd decorator.CmdHandler[SubmitSolParams]

type SubmitSolParams struct {
	UUID        uuid.UUID
	Submission  string
	ProgrLangID string
	TaskShortID string
	AuthorUUID  uuid.UUID
}

type SubmitSolCmdHandler struct {
	DoesUserExist    func(ctx context.Context, uuid uuid.UUID) (bool, error)
	GetTask          func(ctx context.Context, shortId string) (tasksrvc.Task, error)
	StoreSubm        func(ctx context.Context, subm subm.Subm) error
	StoreEval        func(ctx context.Context, eval subm.Eval) error
	BcastSubmCreated func(subm subm.Subm)
	EnqueueEvalExec  func(ctx context.Context, eval subm.Eval, srcCode string, prLangId string) error
}

func (h SubmitSolCmdHandler) Handle(ctx context.Context, p SubmitSolParams) error {
	if len(p.Submission) > 64*1024 { // 64 KB
		return submerror.ErrSubmissionTooLong(64)
	}

	exists, err := h.DoesUserExist(ctx, p.AuthorUUID)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return submerror.ErrUserNotFound()
	}

	l, err := planglist.GetProgrLangById(p.ProgrLangID)
	if err != nil {
		return fmt.Errorf("failed to get progr lang: %w", err)
	}

	t, err := h.GetTask(ctx, p.TaskShortID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	evalUuid := uuid.New()

	entity := subm.Subm{
		UUID:         p.UUID,
		Content:      p.Submission,
		AuthorUUID:   p.AuthorUUID,
		TaskShortID:  p.TaskShortID,
		LangShortID:  l.ID,
		CurrEvalUUID: evalUuid,
		CreatedAt:    time.Now(),
	}
	eval := subm.NewEval(evalUuid, entity.UUID, t)

	err = h.StoreEval(ctx, eval)
	if err != nil {
		return fmt.Errorf("failed to store evaluation: %w", err)
	}

	err = h.StoreSubm(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to store submission: %w", err)
	}

	h.BcastSubmCreated(entity)

	err = h.EnqueueEvalExec(ctx, eval, entity.Content, l.ID)
	if err != nil {
		return fmt.Errorf("failed to enqueue execution of submission: %w", err)
	}

	return nil
}
