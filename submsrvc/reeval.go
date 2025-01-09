package submsrvc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
)

func (s *SubmissionSrvc) ReevalSubm(ctx context.Context, submUuid uuid.UUID) error {
	subm, err := s.GetSubm(ctx, submUuid)
	if err != nil {
		return err
	}

	t, err := s.taskSrvc.GetTask(ctx, subm.Task.ShortID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// enqueue evaluation into sqs for testing
	evalUuid, err := s.execSrvc.Enqueue(execsrvc.CodeWithLang{
		SrcCode: subm.Content,
		LangId:  subm.Lang.ShortID,
	}, s.evalReqTests(&t), execsrvc.TesterParams{
		CpuMs:      t.CpuMillis(),
		MemKiB:     t.MemoryKiB(),
		Checker:    t.CheckerPtr(),
		Interactor: t.InteractorPtr(),
	})
	if err != nil {
		return fmt.Errorf("failed to enqueue evaluation: %w", err)
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
	}
	if len(t.TestGroups) > 0 {
		scoreUnit = ScoreUnitTestGroup
	}

	ch, err := s.execSrvc.Listen(evalUuid)
	if err != nil {
		return fmt.Errorf("failed to listen for evaluation updates: %w", err)
	}

	eval := Evaluation{
		UUID:       evalUuid,
		Stage:      execsrvc.StageWaiting,
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
		SubmUUID:   submUuid,
	}
	s.inMem[submUuid] = eval
	go s.handleUpdates(eval, ch)

	return nil
}
