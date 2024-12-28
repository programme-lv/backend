package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/backend/tasksrvc"
)

type CreateSubmissionParams struct {
	Submission string
	Username   string
	ProgLangID string
	TaskCodeID string
}

func (s *SubmissionSrvc) CreateSubmission(ctx context.Context,
	params *CreateSubmissionParams) (*Submission, error) {

	if len(params.Submission) > 64*1024 { // 64 KB
		return nil, ErrSubmissionTooLong(64)
	}

	u, err := s.userSrvc.GetUserByUsername(ctx, params.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	l, err := planglist.GetProgrammingLanguageById(params.ProgLangID)
	if err != nil {
		return nil, fmt.Errorf("failed to get programming language: %w", err)
	}

	t, err := s.taskSrvc.GetTask(ctx, params.TaskCodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// enqueue evaluation into sqs for testing
	evalUuid, err := s.evalSrvc.Enqueue(evalsrvc.CodeWithLang{
		SrcCode: params.Submission,
		LangId:  params.ProgLangID,
	}, s.evalReqTests(&t), evalsrvc.TesterParams{
		CpuMs:      t.CpuMillis(),
		MemKiB:     t.MemoryKiB(),
		Checker:    t.CheckerPtr(),
		Interactor: t.InteractorPtr(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue evaluation: %w", err)
	}

	subtasks := []Subtask{}
	for _, subtask := range t.Subtasks {
		subtasks = append(subtasks, Subtask{
			Points:      subtask.Score,
			Description: subtask.Descriptions["lv"],
			StTests:     subtask.TestIDs,
		})
	}

	testgroups := []TestGroup{}
	for i, tg := range t.TestGroups {
		testgroups = append(testgroups, TestGroup{
			Points:   tg.Points,
			Subtasks: t.FindTestGroupSubtasks(i + 1),
			TgTests:  tg.TestIDs,
		})
	}

	tests := []Test{}
	for range t.Tests {
		tests = append(tests, Test{})
	}

	scoreUnit := ScoreUnitTest
	if len(t.Subtasks) > 0 {
		scoreUnit = ScoreUnitSubtask
	} else if len(t.TestGroups) > 0 {
		scoreUnit = ScoreUnitTestGroup
	}

	ch, err := s.evalSrvc.Listen(evalUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to listen for evaluation updates: %w", err)
	}

	submUuid := uuid.New()
	subm := Submission{
		UUID:    submUuid,
		Content: params.Submission,
		Author:  Author{UUID: u.UUID, Username: u.Username},
		Task:    TaskRef{ShortID: t.ShortId, FullName: t.FullName},
		Lang:    PrLang{ShortID: l.ID, Display: l.FullName, MonacoID: l.MonacoId},
		CurrEval: Evaluation{
			UUID:       evalUuid,
			Stage:      evalsrvc.StageWaiting,
			ScoreUnit:  scoreUnit,
			Error:      nil,
			Subtasks:   subtasks,
			Groups:     testgroups,
			Tests:      tests,
			Checker:    t.CheckerPtr(),
			Interactor: t.InteractorPtr(),
			CpuLimMs:   t.CpuMillis(),
			MemLimKiB:  t.MemoryKiB(),
			CreatedAt:  time.Now(),
		},
		CreatedAt: time.Now(),
	}

	s.broadcastNewSubmCreated(subm)
	go s.handleUpdates(subm, ch)

	return &subm, nil
}

func (s *SubmissionSrvc) handleUpdates(subm Submission, ch <-chan evalsrvc.Event) {
	timer := time.After(30 * time.Second)
	eval := subm.CurrEval
	for {
		select {
		case update, ok := <-ch:
			if !ok {
				return
			}
			slog.Info("received update from eval srvc", "update", update)
			newEval := applyUpdate(eval, update)
			if !reflect.DeepEqual(newEval, eval) { // i don't give a ****
				s.broadcastSubmEvalUpdate(&EvalUpdate{
					SubmUuid: subm.UUID,
					Eval:     &newEval,
				})
				eval = newEval
			}
			final := false
			final = final || update.Type() == evalsrvc.InternalServerErrorType
			final = final || update.Type() == evalsrvc.CompilationErrorType
			final = final || update.Type() == evalsrvc.FinishedTestingType
			if final {
				subm.CurrEval = newEval
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
