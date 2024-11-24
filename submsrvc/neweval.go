package submsrvc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/backend/tasksrvc"
)

// creates a new evaluation, stores it in the database, and returns its UUID
func (s *SubmissionSrvc) InsertNewEvaluation(ctx context.Context,
	task *tasksrvc.Task,
	lang *planglist.ProgrammingLang,
) (uuid.UUID, error) {
	evalUuid, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	eval := model.Evaluations{
		EvalUUID:           evalUuid,
		EvaluationStage:    "waiting",
		ScoringMethod:      determineScoringMethod(task),
		CPUTimeLimitMillis: int32(task.CpuTimeLimSecs * 1000),
		MemLimitKibiBytes:  int32(float64(task.MemLimMegabytes) * 976.5625),
		TestlibCheckerCode: task.CheckerPtr(),
		TestlibInteractor:  task.InteractorPtr(),
		LangID:             lang.ID,
		LangName:           lang.FullName,
		LangCodeFname:      lang.CodeFilename,
		LangCompCmd:        lang.CompileCmd,
		LangCompFname:      lang.CompiledFilename,
		LangExecCmd:        lang.ExecuteCmd,
		CreatedAt:          time.Now(),
	}

	evalTests := make([]model.EvaluationTests, len(task.Tests))
	for i, test := range task.Tests {
		testID := i + 1

		var subtasksStrPtr *string
		subtasks := task.FindSubtasksWithTest(testID)
		if len(subtasks) > 0 {
			subtaskArray := make([]string, len(subtasks))
			for i, subtask := range subtasks {
				subtaskArray[i] = fmt.Sprintf("%d", subtask.ID)
			}
			subtaskString := fmt.Sprintf("{%s}", strings.Join(subtaskArray, ","))
			subtasksStrPtr = &subtaskString
		}

		var testGroupsStrPtr *string
		testGroups := task.FindTestGroupsWithTest(testID)
		if len(testGroups) > 0 {
			testGroupArray := make([]string, len(testGroups))
			for i, testGroup := range testGroups {
				testGroupArray[i] = fmt.Sprintf("%d", testGroup.ID)
			}
			testGroupString := fmt.Sprintf("{%s}", strings.Join(testGroupArray, ","))
			testGroupsStrPtr = &testGroupString
		}

		evalTest := model.EvaluationTests{
			EvalUUID:        evalUuid,
			TestID:          int32(testID),
			FullInputS3URL:  test.FullInputS3URL(),
			FullAnswerS3URL: test.FullAnswerS3URL(),
			Subtasks:        subtasksStrPtr,
			Testgroups:      testGroupsStrPtr,
		}

		evalTests[i] = evalTest
	}

	evalSubtasks := make([]model.EvaluationSubtasks, len(task.Subtasks))
	for i, subtask := range task.Subtasks {
		subtaskID := i + 1
		var description *string
		if lvDesc, ok := subtask.Descriptions["lv"]; ok {
			description = &lvDesc
		}

		evalSubtasks[i] = model.EvaluationSubtasks{
			EvalUUID:      evalUuid,
			SubtaskID:     int32(subtaskID),
			SubtaskPoints: int32(subtask.Score),
			Accepted:      0,
			Wrong:         0,
			Untested:      int32(len(subtask.TestIDs)),
			Description:   description,
		}
	}

	evalTestGroups := make([]model.EvaluationTestgroups, len(task.TestGroups))
	for i, testGroup := range task.TestGroups {
		testGroupID := i + 1
		subtasks := task.FindTestGroupSubtasks(testGroupID)
		subtasksStr := make([]string, len(subtasks))
		for j, subtask := range subtasks {
			subtasksStr[j] = fmt.Sprintf("%d", subtask)
		}
		subtasksArray := fmt.Sprintf("{%s}", strings.Join(subtasksStr, ","))

		evalTestGroups[i] = model.EvaluationTestgroups{
			EvalUUID:          evalUuid,
			TestgroupID:       int32(testGroupID),
			Accepted:          0,
			Wrong:             0,
			Untested:          int32(len(testGroup.TestIDs)),
			TestgroupPoints:   int32(testGroup.Points),
			StatementSubtasks: &subtasksArray,
		}
	}

	evalTestSet := &model.EvaluationTestset{
		EvalUUID: evalUuid,
		Accepted: 0,
		Wrong:    0,
		Untested: int32(len(task.Tests)),
	}

	tx, err := s.postgres.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = table.Evaluations.
		INSERT(table.Evaluations.AllColumns).
		MODEL(eval).
		Exec(tx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert evaluation: %w", err)
	}

	insertStmt := table.EvaluationTests.
		INSERT(table.EvaluationTests.AllColumns)
	for _, test := range evalTests {
		insertStmt = insertStmt.MODEL(&test)
	}
	_, err = insertStmt.Exec(tx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert evaluation tests: %w", err)
	}

	insertStmt = table.EvaluationSubtasks.
		INSERT(table.EvaluationSubtasks.AllColumns)
	for _, subtask := range evalSubtasks {
		insertStmt = insertStmt.MODEL(&subtask)
	}
	_, err = insertStmt.Exec(tx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert evaluation subtasks: %w", err)
	}

	insertStmt = table.EvaluationTestgroups.
		INSERT(table.EvaluationTestgroups.AllColumns)
	for _, testGroup := range evalTestGroups {
		insertStmt = insertStmt.MODEL(&testGroup)
	}
	_, err = insertStmt.Exec(tx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert evaluation test groups: %w", err)
	}

	insertStmt = table.EvaluationTestset.
		INSERT(table.EvaluationTestset.AllColumns).
		MODEL(evalTestSet)
	_, err = insertStmt.Exec(tx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert evaluation test set: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return evalUuid, nil
}
