package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/tasksrvc"
)

func (s *SubmissionSrvc) handleUpdates(eval Evaluation, ch <-chan execsrvc.Event) {
	l := s.logger.With("eval-uuid", eval.UUID)
	wasSaved := false
	for update := range ch {
		l.Info("received eval update", "type", update.Type())
		newEval := applyUpdate(eval, update)
		eval = newEval
		s.inMemLock.Lock()
		isTheCurrentEval := s.inMem[eval.SubmUUID].UUID == eval.UUID
		if isTheCurrentEval {
			s.inMem[eval.SubmUUID] = eval
		}
		s.inMemLock.Unlock()
		if !isTheCurrentEval {
			break
		}
		s.broadcastSubmEvalUpdate(&EvalUpdate{
			SubmUuid: eval.SubmUUID,
			Eval:     newEval,
		})
		final := false
		final = final || update.Type() == execsrvc.InternalServerErrorType
		final = final || update.Type() == execsrvc.CompilationErrorType
		final = final || update.Type() == execsrvc.FinishedTestingType
		if final {
			wasSaved = true
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := s.evalRepo.Store(ctx, eval)
			if err != nil {
				slog.Error("failed to store submission", "error", err)
			}
			err = s.submRepo.AssignEval(ctx, eval.SubmUUID, eval.UUID)
			if err != nil {
				slog.Error("failed to assign evaluation to submission", "error", err)
			}
			s.inMemLock.Lock()
			delete(s.inMem, eval.SubmUUID)
			s.inMemLock.Unlock()
			return
		}
	}
	if !wasSaved {
		s.logger.Error("evaluation was not saved via listening to events", "eval", eval)
	}
}

func applyUpdate(eval Evaluation, update execsrvc.Event) Evaluation {
	switch u := update.(type) {
	case execsrvc.ReceivedSubmission:
	case execsrvc.StartedCompiling:
		eval.Stage = StageCompiling
	case execsrvc.StartedTesting:
		eval.Stage = StageTesting
	case execsrvc.FinishedTesting:
		eval.Stage = StageFinished
	case execsrvc.InternalServerError:
		eval.Stage = StageFinished
		eval.Error = &EvaluationError{
			Type:    ErrorTypeInternal,
			Message: u.ErrorMsg,
		}
	case execsrvc.CompilationError:
		eval.Stage = StageFinished
		eval.Error = &EvaluationError{
			Type:    ErrorTypeCompilation,
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

func (s *SubmissionSrvc) evalReqTests(task *tasksrvc.Task) []execsrvc.TestFile {
	evalReqTests := make([]execsrvc.TestFile, len(task.Tests))
	for i, test := range task.Tests {
		inputKey := fmt.Sprintf("%s.zst", test.InpSha2)
		answerKey := fmt.Sprintf("%s.zst", test.AnsSha2)
		inputS3Url, err := s.tests.PresignedURL(inputKey, 10*time.Hour)
		if err != nil {
			slog.Error("failed to get presigned URL for input", "error", err)
		}
		answerS3Url, err := s.tests.PresignedURL(answerKey, 10*time.Hour)
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
