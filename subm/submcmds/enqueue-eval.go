package submcmds

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/decorator"
	"github.com/programme-lv/backend/tasksrvc"
)

type EnqueueEvalCmd decorator.CmdHandler[EnqueueEvalParams]

func NewEnqueueEvalCmd(
	getSubm func(ctx context.Context, uuid uuid.UUID) (subm.Subm, error),
	getEval func(ctx context.Context, evalUuid uuid.UUID) (subm.Eval, error),
	enqueueExec func(
		subm execsrvc.CodeWithLang,
		tests []execsrvc.TestFile,
		params execsrvc.TesterParams,
	) (uuid.UUID, error),
	listenToExec func(evalUuid uuid.UUID) (<-chan execsrvc.Event, error),
	updateEvalInMem func(eval subm.Eval),
	storeEvalFinal func(ctx context.Context, eval subm.Eval) error,
) EnqueueEvalCmd {
	return enqueueEvalHandler{
		getSubm:         getSubm,
		getEval:         getEval,
		enqueueExec:     enqueueExec,
		listenToExec:    listenToExec,
		updateEvalInMem: updateEvalInMem,
		storeEvalFinal:  storeEvalFinal,
	}
}

type enqueueEvalHandler struct {
	getSubm func(ctx context.Context, uuid uuid.UUID) (subm.Subm, error)
	getEval func(ctx context.Context, evalUuid uuid.UUID) (subm.Eval, error)

	enqueueExec func(
		subm execsrvc.CodeWithLang,
		tests []execsrvc.TestFile,
		params execsrvc.TesterParams,
	) (uuid.UUID, error)

	listenToExec func(evalUuid uuid.UUID) (<-chan execsrvc.Event, error)

	getTaskByShortId func(ctx context.Context, shortId string) (tasksrvc.Task, error)

	getTestDownlUrl func(ctx context.Context, testId string) (string, error)

	updateEvalInMem func(eval subm.Eval) // should also broadcast to listeners
	storeEvalFinal  func(ctx context.Context, eval subm.Eval) error
}

type EnqueueEvalParams struct {
	EvalUUID uuid.UUID
}

func (h enqueueEvalHandler) Handle(ctx context.Context, p EnqueueEvalParams) error {
	eval, err := h.getEval(ctx, p.EvalUUID)
	if err != nil {
		return fmt.Errorf("failed to get evaluation: %w", err)
	}

	subm, err := h.getSubm(ctx, eval.SubmUUID)
	if err != nil {
		return fmt.Errorf("failed to get subm: %w", err)
	}

	t, err := h.getTaskByShortId(ctx, subm.TaskShortID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	execUuid, err := h.enqueueExec(execsrvc.CodeWithLang{
		SrcCode: subm.Content,
		LangId:  subm.LangShortID,
	}, evalReqTests(ctx, &t, h.getTestDownlUrl), execsrvc.TesterParams{
		CpuMs:      t.CpuMillis(),
		MemKiB:     t.MemoryKiB(),
		Checker:    t.CheckerPtr(),
		Interactor: t.InteractorPtr(),
	})
	if err != nil {
		return fmt.Errorf("failed to enqueue evaluation: %w", err)
	}

	ch, err := h.listenToExec(execUuid)
	if err != nil {
		return fmt.Errorf("failed to listen for evaluation updates: %w", err)
	}

	go h.handleExecUpdates(slog.Default(), eval, ch)

	return nil
}

func evalReqTests(
	ctx context.Context,
	task *tasksrvc.Task,
	getTestDownlUrl func(ctx context.Context, testId string) (string, error),
) []execsrvc.TestFile {
	evalReqTests := make([]execsrvc.TestFile, len(task.Tests))
	for i, test := range task.Tests {
		inputKey := fmt.Sprintf("%s.zst", test.InpSha2)
		answerKey := fmt.Sprintf("%s.zst", test.AnsSha2)
		inputS3Url, err := getTestDownlUrl(ctx, inputKey)
		if err != nil {
			slog.Error("failed to get presigned URL for input", "error", err)
		}
		answerS3Url, err := getTestDownlUrl(ctx, answerKey)
		if err != nil {
			slog.Error("failed to get presigned URL for answer", "error", err)
		}
		evalReqTests[i] = execsrvc.TestFile{
			InSha256:    &test.InpSha2,
			AnsSha256:   &test.AnsSha2,
			InDownlUrl:  &inputS3Url,
			AnsDownlUrl: &answerS3Url,
		}
	}
	return evalReqTests
}

func (h enqueueEvalHandler) handleExecUpdates(logger *slog.Logger, eval subm.Eval, ch <-chan execsrvc.Event) {
	l := logger.With("eval-uuid", eval.UUID)
	wasSaved := false
	for update := range ch {
		l.Info("received eval update", "type", update.Type())
		eval = applyExecEventToEval(eval, update)
		h.updateEvalInMem(eval)
		final := false
		final = final || update.Type() == execsrvc.InternalServerErrorType
		final = final || update.Type() == execsrvc.CompilationErrorType
		final = final || update.Type() == execsrvc.FinishedTestingType
		if final {
			wasSaved = true
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := h.storeEvalFinal(ctx, eval)
			if err != nil {
				slog.Error("failed to store evaluation", "error", err)
			}
			return
		}
	}
	if !wasSaved {
		slog.Error("evaluation was not saved via listening to events", "eval", eval)
	}
}

func applyExecEventToEval(eval subm.Eval, event execsrvc.Event) subm.Eval {
	switch u := event.(type) {
	case execsrvc.ReceivedSubmission:
	case execsrvc.StartedCompiling:
		eval.Stage = subm.EvalStageCompiling
	case execsrvc.StartedTesting:
		eval.Stage = subm.EvalStageTesting
	case execsrvc.FinishedTesting:
		eval.Stage = subm.EvalStageFinished
	case execsrvc.InternalServerError:
		eval.Stage = subm.EvalStageFinished
		eval.Error = &subm.EvalError{
			Type:    subm.ErrorTypeInternal,
			Message: u.ErrorMsg,
		}
	case execsrvc.CompilationError:
		eval.Stage = subm.EvalStageFinished
		eval.Error = &subm.EvalError{
			Type:    subm.ErrorTypeCompilation,
			Message: u.ErrorMsg,
		}
	case execsrvc.ReachedTest:
		eval.Tests[u.TestId-1].Reached = true
	case execsrvc.FinishedTest:
		eval.Tests[u.TestID-1].Finished = true
		if u.Subm != nil {
			if u.Subm.ExitCode != 0 {
				eval.Tests[u.TestID-1].Re = true
			} else if u.Subm.StdErr != "" {
				eval.Tests[u.TestID-1].Re = true
			} else if u.Subm.Signal != nil {
				eval.Tests[u.TestID-1].Re = true
			} else if u.Subm.CpuMs > int64(eval.CpuLimMs) {
				eval.Tests[u.TestID-1].Tle = true
			} else if u.Subm.MemKiB > int64(eval.MemLimKiB) {
				eval.Tests[u.TestID-1].Mle = true
			} else if u.Checker != nil {
				if u.Checker.ExitCode == 0 {
					eval.Tests[u.TestID-1].Ac = true
				} else {
					eval.Tests[u.TestID-1].Wa = true
				}
			}
		}
	case execsrvc.IgnoredTest:
		eval.Tests[u.TestId-1].Ig = true
	}
	return eval
}
