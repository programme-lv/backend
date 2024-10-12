package submsrvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/user"
	"golang.org/x/sync/errgroup"
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
		return nil, NewErrSubmissionTooLong(64)
	}

	// Use errgroup to manage concurrent tasks
	var errCtx context.Context
	g, errCtx := errgroup.WithContext(ctx)

	var (
		userResult *user.User
		languages  []ProgrammingLang
		task       *tasksrvc.Task
	)

	// Parallelize fetching user, languages, and task
	g.Go(func() error {
		u, err := s.userSrvc.GetUserByUsername(errCtx,
			&user.GetUserByUsernamePayload{Username: params.Username})
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		userResult = u
		return nil
	})

	g.Go(func() error {
		langs, err := s.ListProgrammingLanguages(errCtx)
		if err != nil {
			return fmt.Errorf("failed to list programming languages: %w", err)
		}
		languages = langs
		return nil
	})

	g.Go(func() error {
		t, err := s.taskSrvc.GetTask(errCtx, params.TaskCodeID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}
		task = &t
		return nil
	})

	// Wait for all initial fetches to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Parse User UUID
	userUuid, err := uuid.Parse(userResult.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user UUID: %w", err)
	}

	// Find the programming language
	var language *ProgrammingLang
	for _, l := range languages {
		if l.ID == params.ProgLangID {
			language = &l
			break
		}
	}
	if language == nil {
		return nil, NewErrInvalidProgLang()
	}

	// Generate UUIDs for evaluation and submission
	evalUuid := uuid.New()
	submUuid := uuid.New()

	// Prepare Evaluation
	eval := s.prepareEvaluation(evalUuid, task, language)

	// Begin a database transaction
	tx, err := s.postgres.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert Evaluation
	if err := s.insertEvaluation(tx, eval); err != nil {
		return nil, fmt.Errorf("failed to insert evaluation: %w", err)
	}

	// Prepare other evaluation-related data
	evalTests := s.prepareEvaluationTests(evalUuid, task)
	subtasks, evalSubtasks, err := s.prepareSubtasks(evalUuid, task)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare subtasks: %w", err)
	}
	testGroups, evalTestGroups, err := s.prepareTestGroups(evalUuid, task)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare test groups: %w", err)
	}
	testSet, evalTestSet := s.prepareTestSet(evalUuid, task)
	submission := s.prepareSubmission(submUuid, params, userUuid, task, language, evalUuid)

	// Use another errgroup for concurrent insertions
	g2 := errgroup.Group{}

	// Insert evaluation tests
	g2.Go(func() error {
		return s.insertEvaluationTests(tx, evalTests)
	})

	// Insert subtasks
	g2.Go(func() error {
		return s.insertSubtasks(tx, evalSubtasks)
	})

	// Insert test groups
	g2.Go(func() error {
		return s.insertTestGroups(tx, evalTestGroups)
	})

	// Insert test set
	g2.Go(func() error {
		return s.insertTestSet(tx, evalTestSet)
	})

	// Wait for concurrent insertions to complete
	if err := g2.Wait(); err != nil {
		return nil, err
	}

	// Insert submission
	if err := s.insertSubmission(tx, submission); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	evalReqTests := make([]ReqTest, len(task.Tests))
	for i, test := range task.Tests {
		inputS3Url := test.FullInputS3URL()
		answerS3Url := test.FullAnswerS3URL()
		evalReqTests[i] = ReqTest{
			ID:            i + 1,
			InputSha256:   test.InpSha2,
			InputS3Url:    &inputS3Url,
			InputContent:  nil,
			InputHttpUrl:  nil,
			AnswerSha256:  test.AnsSha2,
			AnswerS3Url:   &answerS3Url,
			AnswerContent: nil,
			AnswerHttpUrl: nil,
		}
	}
	req := EvalRequest{
		EvalUuid:  evalUuid.String(),
		ResSqsUrl: &s.resSqsUrl,
		Code:      params.Submission,
		Language: Language{
			LangID:        language.ID,
			LangName:      language.FullName,
			CodeFname:     language.CodeFilename,
			CompileCmd:    language.CompileCmd,
			CompiledFname: language.CompiledFilename,
			ExecCmd:       language.ExecuteCmd,
		},
		Tests:     evalReqTests,
		Checker:   *getChecker(task),
		CpuMillis: int(task.CpuTimeLimSecs * 1000),
		MemoryKiB: int(float64(task.MemLimMegabytes) * 976.5625),
	}
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal evaluation request: %w", err)
	}
	_, err = s.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &s.submSqsUrl,
		MessageBody: aws.String(string(jsonReq)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send message to evaluation queue: %w", err)
	}

	// Assemble the Submission response
	res := &Submission{
		UUID:    submUuid,
		Content: params.Submission,
		Author: Author{
			UUID:     userUuid,
			Username: userResult.Username,
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
	}

	s.createNewSubmChan <- res

	return res, nil
}

// InsertEvaluation inserts the prepared evaluation into the database within a transaction.
func (s *SubmissionSrvc) insertEvaluation(tx *sql.Tx, eval *model.Evaluations) error {
	evalInsertStmt := table.Evaluations.
		INSERT(table.Evaluations.AllColumns).
		MODEL(eval)
	_, err := evalInsertStmt.Exec(tx)
	return err
}

// InsertEvaluationTests inserts the prepared evaluation tests into the database within a transaction.
func (s *SubmissionSrvc) insertEvaluationTests(tx *sql.Tx, tests []model.EvaluationTests) error {
	insertStmt := table.EvaluationTests.
		INSERT(table.EvaluationTests.AllColumns)
	for _, test := range tests {
		insertStmt = insertStmt.MODEL(&test)
	}
	_, err := insertStmt.Exec(tx)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation tests: %w", err)
	}
	return nil
}

// InsertSubtasks inserts the prepared evaluation subtasks into the database within a transaction.
func (s *SubmissionSrvc) insertSubtasks(tx *sql.Tx, evalSubtasks []model.EvaluationSubtasks) error {
	insertStmt := table.EvaluationSubtasks.
		INSERT(table.EvaluationSubtasks.AllColumns)
	for _, subtask := range evalSubtasks {
		insertStmt = insertStmt.MODEL(&subtask)
	}
	_, err := insertStmt.Exec(tx)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation subtasks: %w", err)
	}
	return nil
}

// InsertTestGroups inserts the prepared evaluation test groups into the database within a transaction.
func (s *SubmissionSrvc) insertTestGroups(tx *sql.Tx, evalTestGroups []model.EvaluationTestgroups) error {
	insertStmt := table.EvaluationTestgroups.
		INSERT(table.EvaluationTestgroups.AllColumns)
	for _, testGroup := range evalTestGroups {
		insertStmt = insertStmt.MODEL(&testGroup)
	}
	_, err := insertStmt.Exec(tx)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation test groups: %w", err)
	}
	return nil
}

// InsertTestSet inserts the prepared evaluation test set into the database within a transaction.
func (s *SubmissionSrvc) insertTestSet(tx *sql.Tx, evalTestSet *model.EvaluationTestset) error {
	insertStmt := table.EvaluationTestset.
		INSERT(table.EvaluationTestset.AllColumns).
		MODEL(evalTestSet)
	_, err := insertStmt.Exec(tx)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation test set: %w", err)
	}
	return nil
}

// InsertSubmission inserts the prepared submission into the database within a transaction.
func (s *SubmissionSrvc) insertSubmission(tx *sql.Tx, submission *model.Submissions) error {
	insertStmt := table.Submissions.
		INSERT(table.Submissions.AllColumns).
		MODEL(submission)
	_, err := insertStmt.Exec(tx)
	if err != nil {
		return fmt.Errorf("failed to insert submission: %w", err)
	}
	return nil
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
	res := TestlibDefaultChecker
	return &res
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
