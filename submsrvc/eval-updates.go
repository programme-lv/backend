package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/tasksrvc"
)

func (s *SubmissionSrvc) handleUpdates(subm Submission, ch <-chan evalsrvc.Event) {
	timer := time.After(30 * time.Second)
	eval := subm.CurrEval
	l := s.logger.With("eval-uuid", eval.UUID)
	for {
		select {
		case update, ok := <-ch:
			if !ok {
				return
			}
			l.Info("received eval update", "type", update.Type())
			newEval := applyUpdate(eval, update)
			if !reflect.DeepEqual(newEval, eval) { // i don't give a ****
				s.broadcastSubmEvalUpdate(&EvalUpdate{
					SubmUuid: subm.UUID,
					Eval:     &newEval,
				})
				eval = newEval
				subm.CurrEval = newEval
				s.inMem[subm.UUID] = subm
			}
			final := false
			final = final || update.Type() == evalsrvc.InternalServerErrorType
			final = final || update.Type() == evalsrvc.CompilationErrorType
			final = final || update.Type() == evalsrvc.FinishedTestingType
			if final {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				err := s.repo.Store(ctx, subm)
				if err != nil {
					slog.Error("failed to store submission", "error", err)
				}
				return
			}

		case <-timer:
			slog.Warn("evaluation timed out")
			return
		}
	}
}

func applyUpdate(eval Evaluation, update evalsrvc.Event) Evaluation {
	switch u := update.(type) {
	case evalsrvc.ReceivedSubmission:
	case evalsrvc.StartedCompiling:
		eval.Stage = StageCompiling
	case evalsrvc.StartedTesting:
		eval.Stage = StageTesting
	case evalsrvc.FinishedTesting:
		eval.Stage = StageFinished
	case evalsrvc.InternalServerError:
		eval.Stage = StageFinished
		eval.Error = &EvaluationError{
			Type:    ErrorTypeInternal,
			Message: u.ErrorMsg,
		}
	case evalsrvc.CompilationError:
		eval.Stage = StageFinished
		eval.Error = &EvaluationError{
			Type:    ErrorTypeCompilation,
			Message: u.ErrorMsg,
		}
	case evalsrvc.ReachedTest:
		eval.Tests[u.TestId-1].Reached = true
	case evalsrvc.FinishedTest:
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
	case evalsrvc.IgnoredTest:
		eval.Tests[u.TestId-1].Ig = true
	}
	return eval
}

func (s *SubmissionSrvc) evalReqTests(task *tasksrvc.Task) []evalsrvc.TestFile {
	evalReqTests := make([]evalsrvc.TestFile, len(task.Tests))
	for i, test := range task.Tests {
		inputKey := fmt.Sprintf("%s.zst", test.InpSha2)
		answerKey := fmt.Sprintf("%s.zst", test.AnsSha2)
		inputS3Url, err := s.tests.PresignedURL(inputKey, 10*time.Minute)
		if err != nil {
			slog.Error("failed to get presigned URL for input", "error", err)
		}
		answerS3Url, err := s.tests.PresignedURL(answerKey, 10*time.Minute)
		if err != nil {
			slog.Error("failed to get presigned URL for answer", "error", err)
		}
		evalReqTests[i] = evalsrvc.TestFile{
			InSha256:    &test.InpSha2,
			AnsSha256:   &test.AnsSha2,
			InDownlUrl:  &inputS3Url,
			AnsDownlUrl: &answerS3Url,
		}
	}
	return evalReqTests
}
