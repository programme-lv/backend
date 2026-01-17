package srvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/exec"
	"github.com/programme-lv/backend/plang"
	"github.com/programme-lv/backend/subm/domain"
	tasksrvc "github.com/programme-lv/backend/task/srvc"
)

type SubmitSolParams struct {
	UUID        uuid.UUID
	Submission  string
	ProgrLangID string
	TaskShortID string
	AuthorUUID  uuid.UUID
}

func (s *submSrvc) SubmitSol(ctx context.Context, p SubmitSolParams) error {
	submitSolCmd := submitSolCmdHandler{
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
		EnqueueExec:      s.enqueueExecAndListen,
	}

	err := submitSolCmd.Handle(ctx, p)
	if err == nil {
		return nil // Return untyped nil to avoid non-nil interface wrapping nil pointer
	}
	return err
}

type submitSolCmdHandler struct {
	DoesUserExist    func(ctx context.Context, uuid uuid.UUID) (bool, error)
	GetTask          func(ctx context.Context, shortId string) (tasksrvc.Task, *srvcerror.Error)
	StoreSubm        func(ctx context.Context, subm domain.Subm) error
	StoreEval        func(ctx context.Context, eval domain.Eval) error
	BcastSubmCreated func(subm domain.Subm)
	EnqueueExec      func(ctx context.Context, eval domain.Eval, srcCode string, prLangId string) error
}

const MaxSubmLengthKB = 64

func (h submitSolCmdHandler) Handle(ctx context.Context, p SubmitSolParams) *srvcerror.Error {
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
	submEntity := domain.Subm{
		UUID:         p.UUID,
		Content:      p.Submission,
		AuthorUUID:   p.AuthorUUID,
		TaskShortID:  p.TaskShortID,
		LangShortID:  l.ID,
		CurrEvalUUID: evalUuid,
		CreatedAt:    time.Now(),
	}
	eval := domain.NewEval(evalUuid, submEntity.UUID, t)

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

func (s *submSrvc) ReEvalSubm(ctx context.Context, submUuid uuid.UUID) *srvcerror.Error {
	reevalSubmCmd := reEvalSubmHandler{
		GetSubm:     s.submRepo.GetSubm,
		GetTask:     s.taskSrvc.GetTask,
		StoreEval:   s.evalRepo.StoreEval,
		AssignEval:  s.submRepo.AssignEval,
		EnqueueExec: s.enqueueExecAndListen,
	}
	return reevalSubmCmd.Handle(ctx, submUuid)
}

type reEvalSubmHandler struct {
	// get persisted submission entity by uuid
	GetSubm func(ctx context.Context, submUuid uuid.UUID) (domain.Subm, error)

	// get persisted task entity by short id
	GetTask func(ctx context.Context, shortId string) (tasksrvc.Task, *srvcerror.Error)

	// persist evaluation entity
	StoreEval func(ctx context.Context, eval domain.Eval) error

	// assign evaluation to submission
	AssignEval func(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error

	// enqueue evaluation for corresponding submission execution by tester
	EnqueueExec func(ctx context.Context, eval domain.Eval, srcCode string, prLangId string) error
}

func (h reEvalSubmHandler) Handle(ctx context.Context, submUuid uuid.UUID) *srvcerror.Error {
	log := ctxlog.FromContext(ctx).With("handler", "re eval subm")
	ctx = ctxlog.WithLogger(ctx, log)

	subm, getSubmErr := h.GetSubm(ctx, submUuid)
	if getSubmErr != nil {
		action := "get subm"
		log.Error(action, "subm_uuid", submUuid, "error", getSubmErr)
		return srvcerror.InternalServerError()
	}

	t, getTaskErr := h.GetTask(ctx, subm.TaskShortID)
	if getTaskErr != nil {
		action := "get task"
		log.Error(action, "task_short_id", subm.TaskShortID, "error", getTaskErr)
		return srvcerror.InternalServerError()
	}

	evalUuid := uuid.New()
	eval := domain.NewEval(evalUuid, subm.UUID, t)

	storeEvalErr := h.StoreEval(ctx, eval)
	if storeEvalErr != nil {
		action := "store eval"
		log.Error(action, "eval_uuid", evalUuid, "error", storeEvalErr)
		return srvcerror.InternalServerError()
	}

	assignEvalErr := h.AssignEval(ctx, subm.UUID, evalUuid)
	if assignEvalErr != nil {
		action := "assign eval to subm"
		log.Error(action, "subm_uuid", subm.UUID, "eval_uuid", evalUuid, "error", assignEvalErr)
		return srvcerror.InternalServerError()
	}

	enqueueExecErr := h.EnqueueExec(ctx, eval, subm.Content, subm.LangShortID)
	if enqueueExecErr != nil {
		action := "enqueue execution"
		log.Error(action, "eval_uuid", evalUuid, "error", enqueueExecErr)
		return srvcerror.InternalServerError()
	}

	return nil
}

func (s *submSrvc) enqueueExecAndListen(ctx context.Context, eval domain.Eval, srcCode string, prLangId string) error {
	log := ctxlog.FromContext(ctx)

	// Add eval to in-progress map before enqueueing
	s.inProgrEval[eval.UUID] = eval

	err := s.execSrvc.Enqueue(
		ctx,
		eval.UUID,
		srcCode,
		prLangId,
		constructExecEnqueueTests(ctx, eval, s.taskSrvc.GetTestDownlUrl),
		exec.TestingParams{
			CpuMs:      eval.CpuLimMs,
			MemKiB:     eval.MemLimKiB,
			Checker:    eval.Checker,
			Interactor: eval.Interactor,
		},
	)
	if err != nil {
		delete(s.inProgrEval, eval.UUID) // remove from map if enqueue fails

		action := "enqueue execution"
		log.Error(action, "eval_uuid", eval.UUID, "error", err)
		return srvcerror.InternalServerError()
	}

	ch, err := s.execSrvc.Listen(ctx, eval.UUID)
	if err != nil {
		delete(s.inProgrEval, eval.UUID) // remove from map if listen fails

		action := "subscribe to execution"
		log.Error(action, "eval_uuid", eval.UUID, "error", err)
		return srvcerror.InternalServerError()
	}

	// create a new background context for the event processing goroutine
	processCtx := context.Background()
	go func(execEvCh <-chan exec.Event) {
		for ev := range execEvCh {
			err := procExecEvCmdHandler{
				StoreEval:     s.evalRepo.StoreEval,
				BcastEvalUpd:  s.broadcastEvalUpdate,
				GetEvalByUuid: s.evalRepo.GetEval,
				InProgrEval:   s.inProgrEval,
			}.Handle(processCtx, procExecEvParams{
				Eval:  eval,
				Event: ev,
			})
			if err != nil {
				action := "process execution event"
				log.Error(action, "eval_uuid", eval.UUID, "error", err)
				continue
			}
		}
	}(ch)

	return nil
}

func constructExecEnqueueTests(
	ctx context.Context,
	eval domain.Eval,
	getTestDownlUrl func(ctx context.Context, testFileSha256 string) (string, *srvcerror.Error),
) []exec.TestFile {
	log := ctxlog.FromContext(ctx)

	evalReqTests := make([]exec.TestFile, len(eval.Tests))
	for i, test := range eval.Tests {
		inputS3Url, err := getTestDownlUrl(ctx, test.InpSha256)
		if err != nil {
			action := "get download URL for input"
			log.Error(action, "sha256", test.InpSha256, "error", err)
		}
		answerS3Url, err := getTestDownlUrl(ctx, test.AnsSha256)
		if err != nil {
			action := "get download URL for answer"
			log.Error(action, "sha256", test.AnsSha256, "error", err)
		}
		evalReqTests[i] = exec.TestFile{
			InSha256:    &test.InpSha256,
			AnsSha256:   &test.AnsSha256,
			InDownlUrl:  &inputS3Url,
			AnsDownlUrl: &answerS3Url,
		}
	}
	return evalReqTests
}

type procExecEvParams struct {
	Eval  domain.Eval
	Event exec.Event
}

type procExecEvCmdHandler struct {
	StoreEval     func(ctx context.Context, eval domain.Eval) error
	BcastEvalUpd  func(eval domain.Eval)
	GetEvalByUuid func(ctx context.Context, uuid uuid.UUID) (domain.Eval, error)
	InProgrEval   map[uuid.UUID]domain.Eval
}

func (h procExecEvCmdHandler) Handle(ctx context.Context, p procExecEvParams) error {
	log := ctxlog.FromContext(ctx)

	latestEval, ok := h.InProgrEval[p.Eval.UUID]
	if !ok {
		action := "eval not found in in-memory cache"
		log.Error(action, "eval_uuid", p.Eval.UUID)
		return srvcerror.InternalServerError()
	}
	slog.Info("received event", "event", fmt.Sprintf("%+v", p.Event.Type()))

	eval := applyExecEventToEval(latestEval, p.Event)

	final := false
	final = final || p.Event.Type() == exec.InternalServerErrorType
	final = final || p.Event.Type() == exec.CompilationErrorType
	final = final || p.Event.Type() == exec.FinishedTestingType

	if final {
		err := h.StoreEval(ctx, eval)
		if err != nil {
			slog.Error("failed to store evaluation", "error", err)
			return err
		}
		delete(h.InProgrEval, p.Eval.UUID)
	} else {
		finishedTests := 0
		for _, test := range eval.Tests {
			if test.Finished {
				finishedTests++
			}
		}
		slog.Info("test progress", "finished", finishedTests, "total", len(eval.Tests))
		h.InProgrEval[p.Eval.UUID] = eval
	}

	h.BcastEvalUpd(eval)
	return nil
}
