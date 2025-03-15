package submcmd

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/logger"
	"github.com/programme-lv/backend/planglist"
	decorator "github.com/programme-lv/backend/srvccqs"
	subm "github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/submerror"
	"github.com/programme-lv/backend/task/srvc"
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
	GetTask          func(ctx context.Context, shortId string) (srvc.Task, error)
	StoreSubm        func(ctx context.Context, subm subm.Subm) error
	StoreEval        func(ctx context.Context, eval subm.Eval) error
	BcastSubmCreated func(subm subm.Subm)
	EnqueueEvalExec  func(ctx context.Context, eval subm.Eval, srcCode string, prLangId string) error
}

func (h SubmitSolCmdHandler) Handle(ctx context.Context, p SubmitSolParams) error {
	log := logger.FromContext(ctx)
	log.Debug("handling submit solution command", "subm_uuid", p.UUID, "author_uuid", p.AuthorUUID, "task_id", p.TaskShortID)

	if len(p.Submission) > 64*1024 { // 64 KB
		log.Warn("submission too long", "size", len(p.Submission))
		return submerror.ErrSubmissionTooLong(64)
	}

	exists, err := h.DoesUserExist(ctx, p.AuthorUUID)
	if err != nil {
		log.Error("failed to check if user exists", "author_uuid", p.AuthorUUID, "error", err)
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		log.Warn("user not found", "author_uuid", p.AuthorUUID)
		return submerror.ErrUserNotFound()
	}

	l, err := planglist.GetProgrLangById(p.ProgrLangID)
	if err != nil {
		log.Error("failed to get programming language", "prog_lang_id", p.ProgrLangID, "error", err)
		return fmt.Errorf("failed to get progr lang: %w", err)
	}

	t, err := h.GetTask(ctx, p.TaskShortID)
	if err != nil {
		log.Error("failed to get task", "task_id", p.TaskShortID, "error", err)
		return fmt.Errorf("failed to get task: %w", err)
	}

	evalUuid := uuid.New()
	log.Debug("generated evaluation UUID", "eval_uuid", evalUuid)

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
		log.Error("failed to store evaluation", "eval_uuid", evalUuid, "error", err)
		return fmt.Errorf("failed to store evaluation: %w", err)
	}

	err = h.StoreSubm(ctx, entity)
	if err != nil {
		log.Error("failed to store submission", "subm_uuid", p.UUID, "error", err)
		return fmt.Errorf("failed to store submission: %w", err)
	}

	log.Debug("broadcasting submission created", "subm_uuid", p.UUID)
	h.BcastSubmCreated(entity)

	err = h.EnqueueEvalExec(ctx, eval, entity.Content, l.ID)
	if err != nil {
		log.Error("failed to enqueue execution", "eval_uuid", evalUuid, "error", err)
		return fmt.Errorf("failed to enqueue execution of submission: %w", err)
	}

	log.Debug("submission solution handled successfully", "subm_uuid", p.UUID, "eval_uuid", evalUuid)
	return nil
}
