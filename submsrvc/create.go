package submsrvc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/backend/tasksrvc"
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
	// Validate & retrieve USER
	user, err := s.userSrvc.GetUserByUsername(ctx,
		&user.GetUserByUsernamePayload{Username: params.Username})
	if err != nil {
		return nil, err
	}
	userUuid, err := uuid.Parse(user.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user UUID: %w", err)
	}

	// Validate & retrieve PROGRAMMING LANGUAGE
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

	// Validate & retrieve TASK
	task, err := s.taskSrvc.GetTask(ctx, params.TaskCodeID)
	if err != nil {
		return nil, err
	}

	evalUuid := uuid.New()
	submUuid := uuid.New()

	// Prepare and Insert Evaluation
	eval := s.prepareEvaluation(evalUuid, &task, language)
	if err := s.insertEvaluation(eval); err != nil {
		return nil, fmt.Errorf("failed to insert evaluation: %w", err)
	}

	// Prepare and Insert Evaluation Tests
	evalTests := s.prepareEvaluationTests(evalUuid, &task)
	if err := s.insertEvaluationTests(evalTests); err != nil {
		return nil, fmt.Errorf("failed to insert evaluation tests: %w", err)
	}

	// Prepare and Insert Subtasks
	subtasks, evalSubtasks, err := s.prepareSubtasks(evalUuid, &task)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare subtasks: %w", err)
	}
	if err := s.insertSubtasks(evalSubtasks); err != nil {
		return nil, err
	}

	// Prepare and Insert Test Groups
	testGroups, evalTestGroups, err := s.prepareTestGroups(evalUuid, &task)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare test groups: %w", err)
	}
	if err := s.insertTestGroups(evalTestGroups); err != nil {
		return nil, err
	}

	// Prepare and Insert Test Set
	testSet, evalTestSet := s.prepareTestSet(evalUuid, &task)
	if err := s.insertTestSet(evalTestSet); err != nil {
		return nil, err
	}

	// Prepare and Insert Submission
	submission := s.prepareSubmission(submUuid, params, userUuid, &task, language, evalUuid)
	if err := s.insertSubmission(submission); err != nil {
		return nil, err
	}

	// Assemble the Submission response
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

func (s *SubmissionSrvc) prepareEvaluation(
	evalUuid uuid.UUID,
	task *tasksrvc.Task,
	language *ProgrammingLang) *model.Evaluations {

	eval := model.Evaluations{
		EvalUUID:           evalUuid,
		EvaluationStage:    "waiting",
		ScoringMethod:      determineScoringMethod(task),
		CPUTimeLimitMillis: int32(task.CpuTimeLimSecs * 1000),
		MemLimitKibiBytes:  int32(float64(task.MemLimMegabytes) * 976.5625),
		TestlibCheckerCode: getChecker(task),
		LangID:             language.ID,
		LangName:           language.FullName,
		LangCodeFname:      language.CodeFilename,
		LangCompCmd:        language.CompileCmd,
		LangCompFname:      language.CompiledFilename,
		LangExecCmd:        language.ExecuteCmd,
		CreatedAt:          time.Now(),
	}

	return &eval
}

const (
	ScoringMethodTests     = "tests"
	ScoringMethodSubtask   = "subtask"
	ScoringMethodTestgroup = "testgroup"
)

func determineScoringMethod(task *tasksrvc.Task) string {
	if len(task.TestGroups) > 0 {
		return ScoringMethodTestgroup
	}
	if len(task.Subtasks) > 0 {
		return ScoringMethodSubtask
	}
	return ScoringMethodTests
}

func getChecker(task *tasksrvc.Task) *string {
	if task.Checker != "" {
		checker := task.Checker
		return &checker
	}
	return nil
}

func (s *SubmissionSrvc) insertEvaluation(eval *model.Evaluations) error {
	evalInsertStmt := table.Evaluations.
		INSERT(table.Evaluations.AllColumns).
		MODEL(eval)
	_, err := evalInsertStmt.Exec(s.postgres)
	return err
}

func (s *SubmissionSrvc) prepareEvaluationTests(evalUuid uuid.UUID, task *tasksrvc.Task) []model.EvaluationTests {
	// Start Generation Here
	evalTests := []model.EvaluationTests{}
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

		evalTests = append(evalTests, evalTest)
	}

	return evalTests
}

func (s *SubmissionSrvc) insertEvaluationTests(tests []model.EvaluationTests) error {
	evalTestInsertStmt := table.EvaluationTests.
		INSERT(table.EvaluationTests.AllColumns)
	for _, test := range tests {
		evalTestInsertStmt = evalTestInsertStmt.MODEL(&test)
	}
	_, err := evalTestInsertStmt.Exec(s.postgres)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation tests: %w", err)
	}
	return nil
}

// PrepareSubtasks prepares the subtasks for the evaluation.
func (s *SubmissionSrvc) prepareSubtasks(evalUuid uuid.UUID, task *tasksrvc.Task) ([]Subtask, []model.EvaluationSubtasks, error) {
	subtasks := make([]Subtask, len(task.Subtasks))
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

		subtasks[i] = Subtask{
			SubtaskID:   subtaskID,
			Points:      subtask.Score,
			Accepted:    0,
			Wrong:       0,
			Untested:    int(len(subtask.TestIDs)),
			Description: subtask.Descriptions["lv"],
		}
	}

	return subtasks, evalSubtasks, nil
}

// InsertSubtasks inserts the prepared evaluation subtasks into the database.
func (s *SubmissionSrvc) insertSubtasks(evalSubtasks []model.EvaluationSubtasks) error {
	insertStmt := table.EvaluationSubtasks.
		INSERT(table.EvaluationSubtasks.AllColumns)

	for _, subtask := range evalSubtasks {
		insertStmt = insertStmt.MODEL(&subtask)
	}

	_, err := insertStmt.Exec(s.postgres)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation subtasks: %w", err)
	}

	return nil
}

// PrepareTestGroups prepares the test groups for the evaluation.
func (s *SubmissionSrvc) prepareTestGroups(evalUuid uuid.UUID, task *tasksrvc.Task) ([]TestGroup, []model.EvaluationTestgroups, error) {
	testGroups := make([]TestGroup, len(task.TestGroups))
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

		testGroups[i] = TestGroup{
			TestGroupID: testGroupID,
			Points:      testGroup.Points,
			Accepted:    0,
			Wrong:       0,
			Untested:    int(len(testGroup.TestIDs)),
			Subtasks:    subtasks,
		}
	}

	return testGroups, evalTestGroups, nil
}

// InsertTestGroups inserts the prepared evaluation test groups into the database.
func (s *SubmissionSrvc) insertTestGroups(evalTestGroups []model.EvaluationTestgroups) error {
	insertStmt := table.EvaluationTestgroups.
		INSERT(table.EvaluationTestgroups.AllColumns)

	for _, testGroup := range evalTestGroups {
		insertStmt = insertStmt.MODEL(&testGroup)
	}

	_, err := insertStmt.Exec(s.postgres)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation test groups: %w", err)
	}

	return nil
}

// PrepareTestSet prepares the test set for the evaluation.
func (s *SubmissionSrvc) prepareTestSet(evalUuid uuid.UUID, task *tasksrvc.Task) (*TestSet, *model.EvaluationTestset) {
	testSet := &TestSet{
		Accepted: 0,
		Wrong:    0,
		Untested: len(task.Tests),
	}

	evalTestSet := &model.EvaluationTestset{
		EvalUUID: evalUuid,
		Accepted: 0,
		Wrong:    0,
		Untested: int32(len(task.Tests)),
	}

	return testSet, evalTestSet
}

// InsertTestSet inserts the prepared evaluation test set into the database.
func (s *SubmissionSrvc) insertTestSet(evalTestSet *model.EvaluationTestset) error {
	insertStmt := table.EvaluationTestset.
		INSERT(table.EvaluationTestset.AllColumns).
		MODEL(evalTestSet)

	_, err := insertStmt.Exec(s.postgres)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation test set: %w", err)
	}

	return nil
}

// PrepareSubmission prepares the submission model.
func (s *SubmissionSrvc) prepareSubmission(
	submUuid uuid.UUID,
	params *CreateSubmissionParams,
	userUuid uuid.UUID,
	task *tasksrvc.Task,
	language *ProgrammingLang,
	evalUuid uuid.UUID,
) *model.Submissions {
	return &model.Submissions{
		SubmUUID:        submUuid,
		Content:         params.Submission,
		AuthorUUID:      userUuid,
		TaskID:          task.ShortId,
		ProgLangID:      language.ID,
		CurrentEvalUUID: &evalUuid,
		CreatedAt:       time.Now(),
	}
}

// InsertSubmission inserts the prepared submission into the database.
func (s *SubmissionSrvc) insertSubmission(submission *model.Submissions) error {
	insertStmt := table.Submissions.
		INSERT(table.Submissions.AllColumns).
		MODEL(submission)

	_, err := insertStmt.Exec(s.postgres)
	if err != nil {
		return fmt.Errorf("failed to insert submission: %w", err)
	}

	return nil
}
