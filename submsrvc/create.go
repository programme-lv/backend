package submsrvc

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/backend/tasksrvc"
	usersrvc "github.com/programme-lv/backend/usersrvc"
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
		return nil, ErrSubmissionTooLong(64)
	}

	user, lang, task, err := fetchUserLangTask(ctx, s, params)
	if err != nil {
		return nil, err
	}

	submUuid := uuid.New()
	evalUuid, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	// Prepare Evaluation
	eval := s.prepareEvaluation(evalUuid, task, lang)

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

	submission := &model.Submissions{
		SubmUUID:        submUuid,
		Content:         params.Submission,
		AuthorUUID:      user.UUID,
		TaskID:          task.ShortId,
		ProgLangID:      lang.ID,
		CurrentEvalUUID: &evalUuid,
		CreatedAt:       time.Now(),
	}

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

	s.evalUuidToSubmUuid.Store(evalUuid, submUuid)

	req := evalsrvc.NewEvalParams{
		Code:       params.Submission,
		Tests:      evalReqTests(task),
		Checker:    task.CheckerPtr(),
		Interactor: task.InteractorPtr(),
		CpuMs:      task.CpuMillis(),
		MemKiB:     task.MemoryKiB(),
		LangId:     params.ProgLangID,
	}

	_, err = s.evalSrvc.EnqueueOld(req, evalUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue evaluation: %w", err)
	}

	// Assemble the Submission response
	res := &Submission{
		UUID:    submUuid,
		Content: params.Submission,
		Author: Author{
			UUID:     user.UUID,
			Username: user.Username,
		},
		Task: Task{
			ShortID:  task.ShortId,
			FullName: task.FullName,
		},
		Lang: Lang{
			ShortID:  lang.ID,
			Display:  lang.FullName,
			MonacoID: lang.MonacoId,
		},
		CreatedAt: submission.CreatedAt,
		CurrEval: Evaluation{
			UUID:      evalUuid,
			Stage:     eval.EvaluationStage,
			CreatedAt: eval.CreatedAt,
			Subtasks:  subtasks,
			Groups:    testGroups,
			TestSet:   testSet,
		},
	}

	s.submCreated <- res

	return res, nil
}

func evalReqTests(task *tasksrvc.Task) []evalsrvc.TestFile {
	evalReqTests := make([]evalsrvc.TestFile, len(task.Tests))
	for i, test := range task.Tests {
		inputS3Url := test.FullInputS3URL()
		answerS3Url := test.FullAnswerS3URL()
		evalReqTests[i] = evalsrvc.TestFile{
			InSha256:    &test.InpSha2,
			AnsSha256:   &test.AnsSha2,
			InDownlUrl:  &inputS3Url,
			AnsDownlUrl: &answerS3Url,
		}
	}
	return evalReqTests
}

func fetchUserLangTask(ctx context.Context, s *SubmissionSrvc,
	params *CreateSubmissionParams) (
	*usersrvc.User, *planglist.ProgrammingLang, *tasksrvc.Task, error) {

	var errCtx context.Context
	g, errCtx := errgroup.WithContext(ctx)

	var (
		user *usersrvc.User
		lang *planglist.ProgrammingLang
		task *tasksrvc.Task
	)
	// Parallelize fetching user, languages, and task
	g.Go(func() error {
		u, err := s.userSrvc.GetUserByUsername(errCtx, params.Username)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		user = u
		return nil
	})

	g.Go(func() error {
		langs, err := planglist.ListProgrammingLanguages()
		if err != nil {
			return fmt.Errorf("failed to list programming languages: %w", err)
		}
		for _, l := range langs {
			if l.ID == params.ProgLangID {
				lang = &l
				break
			}
		}
		if lang == nil {
			return ErrInvalidProgLang(params.ProgLangID)
		}
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
		return nil, nil, nil, err
	}

	return user, lang, task, nil
}

// InsertEvaluation inserts the prepared evaluation into the database within a transaction.
func (s *SubmissionSrvc) insertEvaluation(tx *sql.Tx, eval *model.Evaluations) error {
	if eval == nil {
		return nil
	}
	evalInsertStmt := table.Evaluations.
		INSERT(table.Evaluations.AllColumns).
		MODEL(eval)
	_, err := evalInsertStmt.Exec(tx)
	return err
}

// InsertEvaluationTests inserts the prepared evaluation tests into the database within a transaction.
func (s *SubmissionSrvc) insertEvaluationTests(tx *sql.Tx, tests []model.EvaluationTests) error {
	if len(tests) == 0 {
		return nil
	}
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
	if len(evalSubtasks) == 0 {
		return nil
	}
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
	if len(evalTestGroups) == 0 {
		return nil
	}
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
	if evalTestSet == nil {
		return nil
	}
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
	if submission == nil {
		return nil
	}
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
	language *planglist.ProgrammingLang) *model.Evaluations {

	eval := model.Evaluations{
		EvalUUID:           evalUuid,
		EvaluationStage:    "waiting",
		ScoringMethod:      determineScoringMethod(task),
		CPUTimeLimitMillis: int32(task.CpuTimeLimSecs * 1000),
		MemLimitKibiBytes:  int32(float64(task.MemLimMegabytes) * 976.5625),
		TestlibCheckerCode: task.CheckerPtr(),
		TestlibInteractor:  task.InteractorPtr(),
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
			GroupID:  testGroupID,
			Points:   testGroup.Points,
			Accepted: 0,
			Wrong:    0,
			Untested: int(len(testGroup.TestIDs)),
			Subtasks: subtasks,
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
