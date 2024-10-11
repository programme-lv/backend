package submsrvc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/backend/user"
)

type CreateSubmissionParams struct {
	Submission string
	Username   string
	ProgLangID string
	TaskCodeID string
}

func (s *SubmissionSrvc) CreateSubmission(ctx context.Context,
	params *CreateSubmissionParams) (*Submission, error) {
	// validate & retrieve USER
	user, err := s.userSrvc.GetUserByUsername(ctx,
		&user.GetUserByUsernamePayload{Username: params.Username})
	if err != nil {
		return nil, err
	}
	userUuid, err := uuid.Parse(user.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user UUID: %w", err)
	}

	// validate & retrieve PROGRAMMING LANGUAGE
	languages, err := s.ListProgrammingLanguages(ctx)
	if err != nil {
		return nil, err
	}
	var language *ProgrammingLang
	for _, l := range languages {
		if l.ID == params.ProgLangID {
			language = &l
			break
		}
	}

	if language == nil {
		return nil, fmt.Errorf("programming language not found")
	}

	// validate & retrieve TASK
	task, err := s.taskSrvc.GetTask(ctx, params.TaskCodeID)
	if err != nil {
		return nil, err
	}

	scoringMethod := "tests"
	if len(task.Subtasks) > 0 {
		scoringMethod = "subtask"
	}
	if len(task.TestGroups) > 0 {
		scoringMethod = "testgroup"
	}

	evalUuid := uuid.New()
	submUuid := uuid.New()

	checker := task.Checker
	if checker == "" {
		checker = TestlibDefaultChecker
	}

	eval := model.Evaluations{
		EvalUUID:           evalUuid,
		EvaluationStage:    "waiting",
		ScoringMethod:      scoringMethod,
		CPUTimeLimitMillis: int32(task.CpuTimeLimSecs * 1000),
		MemLimitKibiBytes:  int32(float64(task.MemLimMegabytes) * 976.5625),
		TestlibCheckerCode: &checker,
		LangID:             language.ID,
		LangName:           language.FullName,
		LangCodeFname:      language.CodeFilename,
		LangCompCmd:        language.CompileCmd,
		LangCompFname:      language.CompiledFilename,
		LangExecCmd:        language.ExecuteCmd,
		CreatedAt:          time.Now(),
	}

	evalInsertStmt := table.Evaluations.
		INSERT(table.Evaluations.AllColumns).
		MODEL(&eval)

	_, err = evalInsertStmt.Exec(s.postgres)
	if err != nil {
		return nil, fmt.Errorf("failed to insert evaluation: %w", err)
	}

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

		evalTestInsertStmt := table.EvaluationTests.
			INSERT(table.EvaluationTests.AllColumns).
			MODEL(&evalTest)

		_, err = evalTestInsertStmt.Exec(s.postgres)
		if err != nil {
			return nil, fmt.Errorf("failed to insert evaluation test: %w", err)
		}

		// TODO: batch insert the test results
	}

	var subtasks []Subtask
	var testGroups []TestGroup
	var testSet *TestSet

	subtasks = make([]Subtask, len(task.Subtasks))
	for i, subtask := range task.Subtasks {
		subtaskID := i + 1
		var description *string
		if lvDesc, ok := subtask.Descriptions["lv"]; ok {
			description = &lvDesc
		}
		evalScoringSubtask := model.EvaluationSubtasks{
			EvalUUID:      evalUuid,
			SubtaskID:     int32(subtaskID),
			SubtaskPoints: int32(subtask.Score),
			Accepted:      0,
			Wrong:         0,
			Untested:      int32(len(subtask.TestIDs)),
			Description:   description,
		}

		evalScoringSubtaskInsertStmt := table.EvaluationSubtasks.
			INSERT(table.EvaluationSubtasks.AllColumns).
			MODEL(&evalScoringSubtask)

		_, err = evalScoringSubtaskInsertStmt.Exec(s.postgres)
		if err != nil {
			return nil, fmt.Errorf("failed to insert evaluation scoring subtask: %w", err)
		}

		subtasks[i] = Subtask{
			SubtaskID:   subtaskID,
			Points:      subtask.Score,
			Accepted:    0,
			Wrong:       0,
			Untested:    int(len(subtask.TestIDs)),
			Description: subtask.Descriptions["lv"],
		}
	}

	testGroups = make([]TestGroup, len(task.TestGroups))
	for i, testGroup := range task.TestGroups {
		testGroupID := i + 1
		subtasks := task.FindTestGroupSubtasks(testGroupID)
		subtasksStr := make([]string, len(subtasks))
		for i, subtask := range subtasks {
			subtasksStr[i] = fmt.Sprintf("%d", subtask)
		}
		subtasksArray := fmt.Sprintf("{%s}", strings.Join(subtasksStr, ","))
		evalScoringTestGroup := model.EvaluationTestgroups{
			EvalUUID:          evalUuid,
			TestgroupID:       int32(testGroupID),
			Accepted:          0,
			Wrong:             0,
			Untested:          int32(len(testGroup.TestIDs)),
			TestgroupPoints:   int32(testGroup.Points),
			StatementSubtasks: &subtasksArray,
		}

		evalScoringTestGroupInsertStmt := table.EvaluationTestgroups.
			INSERT(table.EvaluationTestgroups.AllColumns).
			MODEL(&evalScoringTestGroup)

		_, err = evalScoringTestGroupInsertStmt.Exec(s.postgres)
		if err != nil {
			return nil, fmt.Errorf("failed to insert evaluation scoring test group: %w", err)
		}

		testGroups[i] = TestGroup{
			TestGroupID: testGroupID,
			Points:      testGroup.Points,
			Accepted:    0,
			Wrong:       0,
			Untested:    int(len(testGroup.TestIDs)),
			Subtasks:    subtasks,
		}
	}

	testSet = &TestSet{
		Accepted: 0,
		Wrong:    0,
		Untested: len(task.Tests),
	}

	evalScoringTestSet := model.EvaluationTestset{
		EvalUUID: evalUuid,
		Accepted: 0,
		Wrong:    0,
		Untested: int32(len(task.Tests)),
	}

	evalScoringTestSetInsertStmt := table.EvaluationTestset.
		INSERT(table.EvaluationTestset.AllColumns).
		MODEL(&evalScoringTestSet)

	_, err = evalScoringTestSetInsertStmt.Exec(s.postgres)
	if err != nil {
		return nil, fmt.Errorf("failed to insert evaluation scoring test set: %w", err)
	}

	submission := model.Submissions{
		SubmUUID:        submUuid,
		Content:         params.Submission,
		AuthorUUID:      userUuid,
		TaskID:          task.ShortId,
		ProgLangID:      language.ID,
		CurrentEvalUUID: &evalUuid,
		CreatedAt:       time.Now(),
	}
	submInsertStmt := table.Submissions.
		INSERT(table.Submissions.AllColumns).
		MODEL(&submission)

	_, err = submInsertStmt.Exec(s.postgres)
	if err != nil {
		return nil, fmt.Errorf("failed to insert submission: %w", err)
	}

	return &Submission{
		UUID:    submUuid,
		Content: params.Submission,
		Author: Author{
			UUID:     userUuid,
			Username: user.Username,
		},
		Task: Task{
			ShortID:  task.ShortId,
			FullName: task.FullName,
		},
		Lang: Lang{
			ShortID:  language.ID,
			Display:  language.FullName,
			MonacoID: language.MonacoId,
		},
		CreatedAt: submission.CreatedAt,
		CurrEval: Evaluation{
			UUID:       evalUuid,
			Stage:      eval.EvaluationStage,
			CreatedAt:  eval.CreatedAt,
			Subtasks:   subtasks,
			TestGroups: testGroups,
			TestSet:    testSet,
		},
	}, nil
}
