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

// InsertNewEvaluation creates a new evaluation, stores it in the database, and returns its UUID
func (s *SubmissionSrvc) InsertNewEvaluation(ctx context.Context,
	task *tasksrvc.Task,
	lang *planglist.ProgrammingLang,
) (uuid.UUID, error) {

	// Generate a new UUID for the evaluation
	evalUUID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	// Create the main evaluation model
	evaluation := buildEvaluationModel(evalUUID, task, lang)

	// Build related evaluation components
	evaluationTests, err := buildEvaluationTests(evalUUID, task)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to build evaluation tests: %w", err)
	}

	evaluationSubtasks := buildEvaluationSubtasks(evalUUID, task)
	evaluationTestGroups, err := buildEvaluationTestGroups(evalUUID, task)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to build evaluation test groups: %w", err)
	}

	evaluationTestSet := buildEvaluationTestSet(evalUUID, task)

	// Begin a new database transaction
	tx, err := s.postgres.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = table.Evaluations.
		INSERT(table.Evaluations.AllColumns).
		MODEL(&evaluation).
		Exec(tx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert evaluation: %w", err)
	}

	insertStmt := table.EvaluationTests.
		INSERT(table.EvaluationTests.AllColumns)
	for _, test := range evaluationTests {
		insertStmt = insertStmt.MODEL(&test)
	}
	_, err = insertStmt.Exec(tx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert evaluation tests: %w", err)
	}

	if len(evaluationSubtasks) > 0 {
		insertStmt = table.EvaluationSubtasks.
			INSERT(table.EvaluationSubtasks.AllColumns)
		for _, subtask := range evaluationSubtasks {
			insertStmt = insertStmt.MODEL(&subtask)
		}
		_, err = insertStmt.Exec(tx)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to insert evaluation subtasks: %w", err)
		}
	}

	if len(evaluationTestGroups) > 0 {
		insertStmt = table.EvaluationTestgroups.
			INSERT(table.EvaluationTestgroups.AllColumns)
		for _, testGroup := range evaluationTestGroups {
			insertStmt = insertStmt.MODEL(&testGroup)
		}
		_, err = insertStmt.Exec(tx)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to insert evaluation test groups: %w", err)
		}
	}

	insertStmt = table.EvaluationTestset.
		INSERT(table.EvaluationTestset.AllColumns).
		MODEL(evaluationTestSet)
	_, err = insertStmt.Exec(tx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert evaluation test set: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return evalUUID, nil
}

// buildEvaluationModel constructs the Evaluations model
func buildEvaluationModel(evalUUID uuid.UUID, task *tasksrvc.Task, lang *planglist.ProgrammingLang) model.Evaluations {
	return model.Evaluations{
		EvalUUID:           evalUUID,
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
}

// buildEvaluationTests constructs a slice of EvaluationTests models
func buildEvaluationTests(evalUUID uuid.UUID, task *tasksrvc.Task) ([]model.EvaluationTests, error) {
	evalTests := make([]model.EvaluationTests, len(task.Tests))
	for i, test := range task.Tests {
		testID := i + 1

		subtaskIDs := make([]int, 0)
		for _, subtask := range task.FindSubtasksWithTest(testID) {
			subtaskIDs = append(subtaskIDs, subtask.ID)
		}
		subtasksStr, err := formatIDs(subtaskIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to format subtasks for test %d: %w", testID, err)
		}

		testGroupIDs := make([]int, 0)
		for _, testGroup := range task.FindTestGroupsWithTest(testID) {
			testGroupIDs = append(testGroupIDs, testGroup.ID)
		}
		testGroupsStr, err := formatIDs(testGroupIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to format test groups for test %d: %w", testID, err)
		}

		evalTests[i] = model.EvaluationTests{
			EvalUUID:        evalUUID,
			TestID:          int32(testID),
			FullInputS3URL:  test.FullInputS3URL(),
			FullAnswerS3URL: test.FullAnswerS3URL(),
			Subtasks:        subtasksStr,
			Testgroups:      testGroupsStr,
		}
	}
	return evalTests, nil
}

// buildEvaluationSubtasks constructs a slice of EvaluationSubtasks models
func buildEvaluationSubtasks(evalUUID uuid.UUID, task *tasksrvc.Task) []model.EvaluationSubtasks {
	evalSubtasks := make([]model.EvaluationSubtasks, len(task.Subtasks))
	for i, subtask := range task.Subtasks {
		subtaskID := i + 1
		description := getDescription(&subtask)

		evalSubtasks[i] = model.EvaluationSubtasks{
			EvalUUID:      evalUUID,
			SubtaskID:     int32(subtaskID),
			SubtaskPoints: int32(subtask.Score),
			Accepted:      0,
			Wrong:         0,
			Untested:      int32(len(subtask.TestIDs)),
			Description:   description,
		}
	}
	return evalSubtasks
}

// buildEvaluationTestGroups constructs a slice of EvaluationTestgroups models
func buildEvaluationTestGroups(evalUUID uuid.UUID, task *tasksrvc.Task) ([]model.EvaluationTestgroups, error) {
	evalTestGroups := make([]model.EvaluationTestgroups, len(task.TestGroups))
	for i, testGroup := range task.TestGroups {
		testGroupID := i + 1

		subtasksStr, err := formatIDs(task.FindTestGroupSubtasks(testGroupID))
		if err != nil {
			return nil, fmt.Errorf("failed to format subtasks for test group %d: %w", testGroupID, err)
		}

		evalTestGroups[i] = model.EvaluationTestgroups{
			EvalUUID:          evalUUID,
			TestgroupID:       int32(testGroupID),
			Accepted:          0,
			Wrong:             0,
			Untested:          int32(len(testGroup.TestIDs)),
			TestgroupPoints:   int32(testGroup.Points),
			StatementSubtasks: subtasksStr,
		}
	}
	return evalTestGroups, nil
}

// buildEvaluationTestSet constructs the EvaluationTestset model
func buildEvaluationTestSet(evalUUID uuid.UUID, task *tasksrvc.Task) *model.EvaluationTestset {
	return &model.EvaluationTestset{
		EvalUUID: evalUUID,
		Accepted: 0,
		Wrong:    0,
		Untested: int32(len(task.Tests)),
	}
}

// formatIDs converts a slice of integers to a Postgres array string pointer
func formatIDs(ids []int) (*string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	strArray := make([]string, len(ids))
	for i, id := range ids {
		strArray[i] = fmt.Sprintf("%d", id)
	}
	formatted := fmt.Sprintf("{%s}", strings.Join(strArray, ","))
	return &formatted, nil
}

// getDescription retrieves the "lv" description if available
func getDescription(subtask *tasksrvc.Subtask) *string {
	if lvDesc, ok := subtask.Descriptions["lv"]; ok {
		return &lvDesc
	}
	return nil
}
