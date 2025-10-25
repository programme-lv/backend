package srvc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/plang"
	subm "github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/srvc/submcmd"
	"github.com/programme-lv/backend/task/srvc"
)

type SubmitSolParams struct {
	UUID        uuid.UUID
	Submission  string
	ProgrLangID string
	TaskShortID string
	AuthorUUID  uuid.UUID
}

func (s *submSrvc) SubmitSol(ctx context.Context, p SubmitSolParams) error {
	submitSolCmd := SubmitSolCmdHandler{
		DoesUserExist: func(ctx context.Context, uuid uuid.UUID) (bool, error) {
			user, err := s.userSrvc.GetUserByUUID(ctx, uuid)
			if err != nil {
				return false, err
			}
			return user.UUID == uuid, nil
		},
		GetTask:          s.taskSrvc.GetTask,
		StoreSubm:        s.submRepo.StoreSubm,
		StoreEval:        s.evalRepo.StoreEval,
		BcastSubmCreated: s.broadcastSubmCreated,
		EnqueueExec:      s.enqueueEvalExecAndListen,
	}

	return submitSolCmd.Handle(ctx, p)
}

type SubmitSolCmdHandler struct {
	DoesUserExist    func(ctx context.Context, uuid uuid.UUID) (bool, error)
	GetTask          func(ctx context.Context, shortId string) (srvc.Task, *srvcerror.Error)
	StoreSubm        func(ctx context.Context, subm subm.Subm) error
	StoreEval        func(ctx context.Context, eval subm.Eval) error
	BcastSubmCreated func(subm subm.Subm)
	EnqueueExec      func(ctx context.Context, eval subm.Eval, srcCode string, prLangId string) error
}

const MaxSubmLengthKB = 64

func (h SubmitSolCmdHandler) Handle(ctx context.Context, p SubmitSolParams) *srvcerror.Error {
	log := ctxlog.FromContext(ctx).With("handler", "submit solution")

	if len(p.Submission) > MaxSubmLengthKB*1024 {
		reason := "submission too long"
		log.Warn(reason, "size", len(p.Submission))
		return ErrSubmTooLong
	}

	exists, err := h.DoesUserExist(ctx, p.AuthorUUID)
	if err != nil {
		action := "check if user exists"
		log.Error(action, "author_uuid", p.AuthorUUID, "error", err)
		return srvcerror.InternalServerError()
	}

	if !exists {
		log.Warn("user not found", "author_uuid", p.AuthorUUID)
		return ErrUserNotFound
	}

	l, getProgrLangErr := plang.GetProgrLangById(p.ProgrLangID)
	if getProgrLangErr != nil {
		action := "get progr lang"
		log.Error(action, "prog_lang_id", p.ProgrLangID, "error", getProgrLangErr)
		return getProgrLangErr
	}

	t, getTaskErr := h.GetTask(ctx, p.TaskShortID)
	if getTaskErr != nil {
		action := "get task"
		log.Error(action, "task_id", p.TaskShortID, "error", getTaskErr)
		return getTaskErr
	}

	evalUuid := uuid.New()
	submEntity := subm.Subm{
		UUID:         p.UUID,
		Content:      p.Submission,
		AuthorUUID:   p.AuthorUUID,
		TaskShortID:  p.TaskShortID,
		LangShortID:  l.ID,
		CurrEvalUUID: evalUuid,
		CreatedAt:    time.Now(),
	}
	eval := subm.NewEval(evalUuid, submEntity.UUID, t)

	err = h.StoreEval(ctx, eval)
	if err != nil {
		action := "store evaluation"
		log.Error(action, "eval_uuid", evalUuid, "error", err)
		return srvcerror.InternalServerError()
	}

	err = h.StoreSubm(ctx, submEntity)
	if err != nil {
		action := "store submission"
		log.Error(action, "subm_uuid", p.UUID, "error", err)
		return srvcerror.InternalServerError()
	}

	h.BcastSubmCreated(submEntity)

	err = h.EnqueueExec(ctx, eval, submEntity.Content, l.ID)
	if err != nil {
		action := "enqueue execution"
		log.Error(action, "eval_uuid", evalUuid, "error", err)
		return srvcerror.InternalServerError()
	}

	return nil
}

func (s *submSrvc) ReEvalSubm(ctx context.Context, submUuid uuid.UUID) error {
	reevalSubmCmd := submcmd.ReEvalSubmHandler{
		GetSubm:     s.submRepo.GetSubm,
		GetTask:     s.taskSrvc.GetTask,
		StoreEval:   s.evalRepo.StoreEval,
		AssignEval:  s.submRepo.AssignEval,
		EnqueueExec: s.enqueueEvalExecAndListen,
	}
	return reevalSubmCmd.Handle(ctx, submUuid)
}
