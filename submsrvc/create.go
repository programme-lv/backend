package submsrvc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/planglist"
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
	}
	if len(t.TestGroups) > 0 {
		scoreUnit = ScoreUnitTestGroup
	}

	ch, err := s.evalSrvc.Listen(evalUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to listen for evaluation updates: %w", err)
	}

	submUuid := uuid.New()

	eval := Evaluation{
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
		SubmUUID:   submUuid,
	}

	entity := SubmissionEntity{
		UUID:        submUuid,
		Content:     params.Submission,
		AuthorUUID:  u.UUID,
		TaskShortID: t.ShortId,
		LangShortID: l.ID,
		CurrEvalID:  uuid.Nil, // this will be set when the evaluation is finished
		CreatedAt:   time.Now(),
	}

	err = s.submRepo.Store(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to store submission: %w", err)
	}

	view := Submission{
		UUID:      submUuid,
		Content:   entity.Content,
		Author:    Author{UUID: entity.AuthorUUID, Username: u.Username},
		Task:      TaskRef{ShortID: entity.TaskShortID, FullName: t.FullName},
		Lang:      PrLang{ShortID: entity.LangShortID, Display: l.FullName, MonacoID: l.MonacoId},
		CurrEval:  &eval,
		CreatedAt: entity.CreatedAt,
	}

	s.inMem[submUuid] = eval
	s.broadcastNewSubmCreated(view)
	go s.handleUpdates(eval, ch)

	return &view, nil
}
