package submsrvc

import (
	"context"

	"github.com/google/uuid"
)

func (s *SubmissionSrvc) ReevalSubm(ctx context.Context, submUuid uuid.UUID) error {
	panic("not implemented")
	// subm, err := s.GetSubm(ctx, submUuid)
	// if err != nil {
	// 	return fmt.Errorf("failed to get submission: %w", err)
	// }

	// l, err := planglist.GetProgrammingLanguageById(subm.Lang.ShortID)
	// if err != nil {
	// 	return fmt.Errorf("failed to get programming language: %w", err)
	// }

	// t, err := s.taskSrvc.GetTask(ctx, subm.Task.ShortID)
	// if err != nil {
	// 	return fmt.Errorf("failed to get task: %w", err)
	// }

	// // enqueue evaluation into sqs for testing
	// evalUuid, err := s.evalSrvc.Enqueue(evalsrvc.CodeWithLang{
	// 	SrcCode: subm.Content,
	// 	LangId:  subm.Lang.ShortID,
	// }, s.evalReqTests(&t), evalsrvc.TesterParams{
	// 	CpuMs:      t.CpuMillis(),
	// 	MemKiB:     t.MemoryKiB(),
	// 	Checker:    t.CheckerPtr(),
	// 	Interactor: t.InteractorPtr(),
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to enqueue evaluation: %w", err)
	// }

	// subtasks := []Subtask{}
	// for _, subtask := range t.Subtasks {
	// 	subtasks = append(subtasks, Subtask{
	// 		Points:      subtask.Score,
	// 		Description: subtask.Descriptions["lv"],
	// 		StTests:     subtask.TestIDs,
	// 	})
	// }

	// testgroups := []TestGroup{}
	// for i, tg := range t.TestGroups {
	// 	testgroups = append(testgroups, TestGroup{
	// 		Points:   tg.Points,
	// 		Subtasks: t.FindTestGroupSubtasks(i + 1),
	// 		TgTests:  tg.TestIDs,
	// 	})
	// }

	// tests := []Test{}
	// for range t.Tests {
	// 	tests = append(tests, Test{})
	// }

	// scoreUnit := ScoreUnitTest
	// if len(t.Subtasks) > 0 {
	// 	scoreUnit = ScoreUnitSubtask
	// }
	// if len(t.TestGroups) > 0 {
	// 	scoreUnit = ScoreUnitTestGroup
	// }

	// ch, err := s.evalSrvc.Listen(evalUuid)
	// if err != nil {
	// 	return fmt.Errorf("failed to listen for evaluation updates: %w", err)
	// }

	// entity := SubmissionEntity{
	// 	UUID:        submUuid,
	// 	Content:     subm.Content,
	// 	AuthorUUID:  subm.Author.UUID,
	// 	TaskShortID: t.ShortId,
	// 	LangShortID: l.ID,
	// 	CurrEvalID:  evalUuid,
	// 	CreatedAt:   subm.CreatedAt,
	// }

	// s.inMem[submUuid] = entity

	// eval := Evaluation{
	// 	UUID:       evalUuid,
	// 	Stage:      evalsrvc.StageWaiting,
	// 	ScoreUnit:  scoreUnit,
	// 	Error:      nil,
	// 	Subtasks:   subtasks,
	// 	Groups:     testgroups,
	// 	Tests:      tests,
	// 	Checker:    t.CheckerPtr(),
	// 	Interactor: t.InteractorPtr(),
	// 	CpuLimMs:   t.CpuMillis(),
	// 	MemLimKiB:  t.MemoryKiB(),
	// 	CreatedAt:  time.Now(),
	// }

	// go s.handleUpdates(eval, ch)

	// return nil
}
