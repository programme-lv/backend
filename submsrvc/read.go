package submsrvc

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/backend/planglist"
)

func (s *SubmissionSrvc) GetFullSubmission(ctx context.Context, submUuid string) (*FullSubmission, error) {
	submUUID, err := uuid.Parse(submUuid)
	if err != nil {
		return nil, fmt.Errorf("invalid submission UUID: %w", err)
	}

	// Build the select statement
	selectSubmStmt := postgres.SELECT(table.Submissions.AllColumns, table.Evaluations.AllColumns, table.EvaluationTestset.AllColumns, table.RuntimeData.AllColumns).
		FROM(
			table.Submissions.
				LEFT_JOIN(table.Evaluations, table.Submissions.CurrentEvalUUID.EQ(table.Evaluations.EvalUUID)).
				LEFT_JOIN(table.EvaluationTestset, table.Evaluations.EvalUUID.EQ(table.EvaluationTestset.EvalUUID)).
				LEFT_JOIN(table.RuntimeData, table.Evaluations.CompileRuntimeID.EQ(table.RuntimeData.ID)),
		).
		WHERE(table.Submissions.SubmUUID.EQ(postgres.UUID(submUUID)))

	// Define the model
	type SubmJoinEvalModel struct {
		model.Submissions
		model.Evaluations
		model.EvaluationTestset
		model.RuntimeData
	}

	var submJoinEval SubmJoinEvalModel
	if err := selectSubmStmt.QueryContext(ctx, s.postgres, &submJoinEval); err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	// Get subtasks
	selectSubtasks := postgres.SELECT(table.EvaluationSubtasks.AllColumns).
		FROM(table.EvaluationSubtasks).
		WHERE(table.EvaluationSubtasks.EvalUUID.EQ(postgres.UUID(submJoinEval.Evaluations.EvalUUID)))

	type SubmSubtaskModel struct {
		model.EvaluationSubtasks
	}

	var subtasks []SubmSubtaskModel
	if err := selectSubtasks.QueryContext(ctx, s.postgres, &subtasks); err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}

	// Get test groups
	selectTestGroups := postgres.SELECT(table.EvaluationTestgroups.AllColumns).
		FROM(table.EvaluationTestgroups).
		WHERE(table.EvaluationTestgroups.EvalUUID.EQ(postgres.UUID(submJoinEval.Evaluations.EvalUUID)))

	type SubmTestGroupModel struct {
		model.EvaluationTestgroups
	}

	var testGroups []SubmTestGroupModel
	if err := selectTestGroups.QueryContext(ctx, s.postgres, &testGroups); err != nil {
		return nil, fmt.Errorf("failed to get test groups: %w", err)
	}

	// Get author username
	usernames, err := s.userSrvc.GetUsernames(ctx, []uuid.UUID{submJoinEval.Submissions.AuthorUUID})
	if err != nil {
		return nil, err
	}

	// Get task full name
	taskFullNames, err := s.taskSrvc.GetTaskFullNames(ctx, []string{submJoinEval.Submissions.TaskID})
	if err != nil {
		return nil, err
	}

	// Get languages
	languages, err := planglist.ListProgrammingLanguages()
	if err != nil {
		return nil, err
	}
	langMap := make(map[string]planglist.ProgrammingLang)
	for _, lang := range languages {
		langMap[lang.ID] = lang
	}

	// Process subtasks
	subtasksList := []Subtask{}
	for _, subtask := range subtasks {
		description := ""
		if subtask.EvaluationSubtasks.Description != nil {
			description = *subtask.EvaluationSubtasks.Description
		}
		subtasksList = append(subtasksList, Subtask{
			SubtaskID:   int(subtask.EvaluationSubtasks.SubtaskID),
			Points:      int(subtask.EvaluationSubtasks.SubtaskPoints),
			Accepted:    int(subtask.EvaluationSubtasks.Accepted),
			Wrong:       int(subtask.EvaluationSubtasks.Wrong),
			Untested:    int(subtask.EvaluationSubtasks.Untested),
			Description: description,
		})
	}

	// Process test groups
	testGroupsList := []TestGroup{}
	for _, testGroup := range testGroups {
		subtaskArrayStrPtr := testGroup.EvaluationTestgroups.StatementSubtasks
		subtaskArray := []int{}
		if subtaskArrayStrPtr != nil {
			subtaskArrayStr := strings.Trim(*subtaskArrayStrPtr, "{}")
			subtaskStrs := strings.Split(subtaskArrayStr, ",")
			for _, subtaskStr := range subtaskStrs {
				subtask, err := strconv.Atoi(subtaskStr)
				if err != nil {
					return nil, fmt.Errorf("failed to convert subtask string to int: %w", err)
				}
				subtaskArray = append(subtaskArray, subtask)
			}
		}
		testGroupsList = append(testGroupsList, TestGroup{
			TestGroupID: int(testGroup.EvaluationTestgroups.TestgroupID),
			Points:      int(testGroup.EvaluationTestgroups.TestgroupPoints),
			Accepted:    int(testGroup.EvaluationTestgroups.Accepted),
			Wrong:       int(testGroup.EvaluationTestgroups.Wrong),
			Untested:    int(testGroup.EvaluationTestgroups.Untested),
			Subtasks:    subtaskArray,
		})
	}

	selectEvalTests := postgres.SELECT(table.EvaluationTests.AllColumns, table.RuntimeData.AS("subm_runtime").AllColumns, table.RuntimeData.AS("checker_runtime").AllColumns).
		FROM(
			table.EvaluationTests.
				LEFT_JOIN(table.RuntimeData.AS("subm_runtime"), table.EvaluationTests.SubmRuntimeID.EQ(table.RuntimeData.AS("subm_runtime").ID)).
				LEFT_JOIN(table.RuntimeData.AS("checker_runtime"), table.EvaluationTests.CheckerRuntimeID.EQ(table.RuntimeData.AS("checker_runtime").ID)),
		).
		WHERE(table.EvaluationTests.EvalUUID.EQ(postgres.UUID(submJoinEval.Evaluations.EvalUUID)))

	type evalTestWithRuntimes struct {
		model.EvaluationTests
		SubmRuntime    model.RuntimeData `alias:"subm_runtime"`
		CheckerRuntime model.RuntimeData `alias:"checker_runtime"`
	}
	var evalTests []evalTestWithRuntimes
	err = selectEvalTests.Query(s.postgres, &evalTests)
	if err != nil {
		return nil, fmt.Errorf("failed to get evaluation tests: %w", err)
	}

	testResults := []EvalTestResult{}
	for _, evalTest := range evalTests {
		subtaskArrayStrPtr := evalTest.Subtasks
		subtaskArray := []int{}
		if subtaskArrayStrPtr != nil {
			subtaskArrayStr := strings.Trim(*subtaskArrayStrPtr, "{}")
			subtaskStrs := strings.Split(subtaskArrayStr, ",")
			for _, subtaskStr := range subtaskStrs {
				subtask, err := strconv.Atoi(subtaskStr)
				if err != nil {
					return nil, fmt.Errorf("failed to convert subtask string to int: %w", err)
				}
				subtaskArray = append(subtaskArray, subtask)
			}
		}

		testGroupArrayStrPtr := evalTest.Testgroups
		testGroupArray := []int{}
		if testGroupArrayStrPtr != nil {
			testGroupArrayStr := strings.Trim(*testGroupArrayStrPtr, "{}")
			testGroupStrs := strings.Split(testGroupArrayStr, ",")
			for _, testGroupStr := range testGroupStrs {
				testGroup, err := strconv.Atoi(testGroupStr)
				if err != nil {
					return nil, fmt.Errorf("failed to convert test group string to int: %w", err)
				}
				testGroupArray = append(testGroupArray, testGroup)
			}
		}
		testResults = append(testResults, EvalTestResult{
			TestId:         int(evalTest.TestID),
			Reached:        evalTest.Reached,
			Ignored:        evalTest.Ignored,
			Finished:       evalTest.Finished,
			InputTrimmed:   evalTest.InputTrimmed,
			AnswerTrimmed:  evalTest.AnswerTrimmed,
			TimeExceeded:   evalTest.SubmRuntime.CPUTimeMillis > int64(submJoinEval.Evaluations.CPUTimeLimitMillis),
			MemoryExceeded: evalTest.SubmRuntime.MemoryKibiBytes > int64(submJoinEval.Evaluations.MemLimitKibiBytes),
			Subtasks:       subtaskArray,
			TestGroups:     testGroupArray,
			SubmRuntime: &RuntimeData{
				CpuMillis:  int(evalTest.SubmRuntime.CPUTimeMillis),
				MemoryKiB:  int(evalTest.SubmRuntime.MemoryKibiBytes),
				WallTime:   int(evalTest.SubmRuntime.WallTimeMillis),
				ExitCode:   int(evalTest.SubmRuntime.ExitCode),
				Stdout:     evalTest.SubmRuntime.Stdout,
				Stderr:     evalTest.SubmRuntime.Stderr,
				ExitSignal: evalTest.SubmRuntime.ExitSignal,
			},
			CheckerRuntime: &RuntimeData{
				CpuMillis:  int(evalTest.CheckerRuntime.CPUTimeMillis),
				MemoryKiB:  int(evalTest.CheckerRuntime.MemoryKibiBytes),
				WallTime:   int(evalTest.CheckerRuntime.WallTimeMillis),
				ExitCode:   int(evalTest.CheckerRuntime.ExitCode),
				Stdout:     evalTest.CheckerRuntime.Stdout,
				Stderr:     evalTest.CheckerRuntime.Stderr,
				ExitSignal: evalTest.CheckerRuntime.ExitSignal,
			},
		})
	}

	var compileRuntime *RuntimeData
	if submJoinEval.RuntimeData.ID != 0 {
		compileRuntime = &RuntimeData{
			CpuMillis:  int(submJoinEval.RuntimeData.CPUTimeMillis),
			MemoryKiB:  int(submJoinEval.RuntimeData.MemoryKibiBytes),
			WallTime:   int(submJoinEval.RuntimeData.WallTimeMillis),
			ExitCode:   int(submJoinEval.RuntimeData.ExitCode),
			Stdout:     submJoinEval.RuntimeData.Stdout,
			Stderr:     submJoinEval.RuntimeData.Stderr,
			ExitSignal: submJoinEval.RuntimeData.ExitSignal,
		}
	}

	// Construct the FullSubmission
	fullSubmission := &FullSubmission{
		Submission: Submission{
			UUID:    submUUID,
			Content: submJoinEval.Submissions.Content,
			Author:  Author{UUID: submJoinEval.Submissions.AuthorUUID, Username: usernames[0]},
			Task:    Task{ShortID: submJoinEval.Submissions.TaskID, FullName: taskFullNames[0]},
			Lang: Lang{
				ShortID:  submJoinEval.Evaluations.LangID,
				Display:  langMap[submJoinEval.Evaluations.LangID].FullName,
				MonacoID: langMap[submJoinEval.Evaluations.LangID].MonacoId,
			},
			CurrEval: Evaluation{
				UUID:       submJoinEval.Evaluations.EvalUUID,
				Stage:      submJoinEval.Evaluations.EvaluationStage,
				CreatedAt:  submJoinEval.Evaluations.CreatedAt,
				Subtasks:   subtasksList,
				TestGroups: testGroupsList,
				TestSet: &TestSet{
					Accepted: int(submJoinEval.EvaluationTestset.Accepted),
					Wrong:    int(submJoinEval.EvaluationTestset.Wrong),
					Untested: int(submJoinEval.EvaluationTestset.Untested),
				},
			},
			CreatedAt: submJoinEval.Submissions.CreatedAt,
		},
		TestResults: testResults,
		EvalDetails: &EvalDetails{
			EvalUuid:           submJoinEval.Evaluations.EvalUUID.String(),
			CreatedAt:          submJoinEval.Evaluations.CreatedAt,
			ErrorMsg:           submJoinEval.Evaluations.ErrorMessage,
			EvalStage:          submJoinEval.Evaluations.EvaluationStage,
			CpuTimeLimitMillis: int(submJoinEval.Evaluations.CPUTimeLimitMillis),
			MemoryLimitKiB:     int(submJoinEval.Evaluations.MemLimitKibiBytes),
			ProgrammingLang: planglist.ProgrammingLang{
				ID:               submJoinEval.Evaluations.LangID,
				FullName:         langMap[submJoinEval.Evaluations.LangID].FullName,
				CodeFilename:     submJoinEval.Evaluations.LangCodeFname,
				CompileCmd:       submJoinEval.Evaluations.LangCompCmd,
				ExecuteCmd:       submJoinEval.Evaluations.LangExecCmd,
				EnvVersionCmd:    "",
				HelloWorldCode:   "",
				MonacoId:         langMap[submJoinEval.Evaluations.LangID].MonacoId,
				CompiledFilename: submJoinEval.Evaluations.LangCompFname,
				Enabled:          langMap[submJoinEval.Evaluations.LangID].Enabled,
			},
			SystemInformation: submJoinEval.Evaluations.SystemInformation,
			CompileRuntime:    compileRuntime,
		},
	}

	return fullSubmission, nil
}

// Start of Selection
func (s *SubmissionSrvc) ListSubmissions(ctx context.Context) ([]*Submission, error) {
	startTotal := time.Now()

	// Fetch submissions
	start := time.Now()
	selectSubmStmt := postgres.SELECT(table.Submissions.AllColumns, table.Evaluations.AllColumns, table.EvaluationTestset.AllColumns).
		FROM(
			table.Submissions.
				LEFT_JOIN(table.Evaluations, table.Submissions.CurrentEvalUUID.EQ(table.Evaluations.EvalUUID)).
				LEFT_JOIN(table.EvaluationTestset, table.Submissions.CurrentEvalUUID.EQ(table.EvaluationTestset.EvalUUID)),
		).ORDER_BY(table.Submissions.CreatedAt.DESC()).LIMIT(30)
	log.Printf("SELECT statement for submissions prepared in %v", time.Since(start))

	start = time.Now()
	type SubmJoinEvalModel struct {
		model.Submissions
		model.Evaluations
		model.EvaluationTestset
	}

	var submJoinEval []SubmJoinEvalModel
	if err := selectSubmStmt.QueryContext(ctx, s.postgres, &submJoinEval); err != nil {
		log.Printf("Error fetching submissions: %v (took %v)", err, time.Since(startTotal))
		return nil, fmt.Errorf("failed to get submissions: %w", err)
	}
	log.Printf("Fetched submissions in %v", time.Since(start))

	// Fetch subtasks
	start = time.Now()
	selectSubtasks := postgres.SELECT(table.Submissions.SubmUUID, table.EvaluationSubtasks.AllColumns).
		FROM(table.Submissions.
			INNER_JOIN(table.EvaluationSubtasks, table.Submissions.CurrentEvalUUID.EQ(table.EvaluationSubtasks.EvalUUID)),
		)
	log.Printf("SELECT statement for subtasks prepared in %v", time.Since(start))

	start = time.Now()
	type SubmSubtaskModel struct {
		model.Submissions
		model.EvaluationSubtasks
	}

	var subtasks []SubmSubtaskModel
	if err := selectSubtasks.QueryContext(ctx, s.postgres, &subtasks); err != nil {
		log.Printf("Error fetching subtasks: %v (took %v)", err, time.Since(startTotal))
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	log.Printf("Fetched subtasks in %v", time.Since(start))

	// Map submissions to subtasks
	start = time.Now()
	submUUIDToSubtask := make(map[uuid.UUID][]SubmSubtaskModel)
	for _, subtask := range subtasks {
		submUUIDToSubtask[subtask.SubmUUID] = append(submUUIDToSubtask[subtask.SubmUUID], subtask)
	}
	log.Printf("Mapped submissions to subtasks in %v", time.Since(start))

	// Fetch test groups
	start = time.Now()
	selectTestGroups := postgres.SELECT(table.Submissions.SubmUUID, table.EvaluationTestgroups.AllColumns).
		FROM(table.Submissions.
			INNER_JOIN(table.EvaluationTestgroups, table.Submissions.CurrentEvalUUID.EQ(table.EvaluationTestgroups.EvalUUID)),
		)
	log.Printf("SELECT statement for test groups prepared in %v", time.Since(start))

	start = time.Now()
	type SubmTestGroupModel struct {
		model.Submissions
		model.EvaluationTestgroups
	}

	var submTestGroups []SubmTestGroupModel
	if err := selectTestGroups.QueryContext(ctx, s.postgres, &submTestGroups); err != nil {
		log.Printf("Error fetching test groups: %v (took %v)", err, time.Since(startTotal))
		return nil, fmt.Errorf("failed to get test groups: %w", err)
	}
	log.Printf("Fetched test groups in %v", time.Since(start))

	// Map submissions to test groups
	start = time.Now()
	submUUIDToTestGroups := make(map[uuid.UUID][]SubmTestGroupModel)
	for _, testGroup := range submTestGroups {
		submUUIDToTestGroups[testGroup.SubmUUID] = append(submUUIDToTestGroups[testGroup.SubmUUID], testGroup)
	}
	log.Printf("Mapped submissions to test groups in %v", time.Since(start))

	// Fetch author UUIDs
	start = time.Now()
	authorUUIDs := make([]uuid.UUID, len(submJoinEval))
	for i, subm := range submJoinEval {
		authorUUIDs[i] = subm.Submissions.AuthorUUID
	}
	log.Printf("Collected author UUIDs in %v", time.Since(start))

	// Fetch usernames
	start = time.Now()
	usernames, err := s.userSrvc.GetUsernames(ctx, authorUUIDs)
	if err != nil {
		log.Printf("Error fetching usernames: %v (took %v)", err, time.Since(startTotal))
		return nil, err
	}
	log.Printf("Fetched usernames in %v", time.Since(start))

	// Fetch task full names
	start = time.Now()
	taskShortIDs := make([]string, len(submJoinEval))
	for i, subm := range submJoinEval {
		taskShortIDs[i] = subm.Submissions.TaskID
	}
	taskFullNames, err := s.taskSrvc.GetTaskFullNames(ctx, taskShortIDs)
	if err != nil {
		log.Printf("Error fetching task full names: %v (took %v)", err, time.Since(startTotal))
		return nil, err
	}
	log.Printf("Fetched task full names in %v", time.Since(start))

	// Fetch programming languages
	start = time.Now()
	languages, err := planglist.ListProgrammingLanguages()
	if err != nil {
		log.Printf("Error fetching programming languages: %v (took %v)", err, time.Since(startTotal))
		return nil, err
	}
	log.Printf("Fetched programming languages in %v", time.Since(start))

	// Map language IDs to names and Monaco IDs
	start = time.Now()
	langIDToFullName := make(map[string]string)
	langIDToMonacoID := make(map[string]string)
	for _, lang := range languages {
		langIDToFullName[lang.ID] = lang.FullName
		langIDToMonacoID[lang.ID] = lang.MonacoId
	}
	log.Printf("Mapped language IDs to names and Monaco IDs in %v", time.Since(start))

	// Process submissions
	start = time.Now()
	submissions := make([]*Submission, len(submJoinEval))
	for i, subm := range submJoinEval {
		// Process subtasks
		subtasks := make([]Subtask, 0)
		subtaskModels, hasSubtasks := submUUIDToSubtask[subm.Submissions.SubmUUID]
		if hasSubtasks {
			for _, subtask := range subtaskModels {
				description := ""
				if subtask.EvaluationSubtasks.Description != nil {
					description = *subtask.EvaluationSubtasks.Description
				}
				subtasks = append(subtasks, Subtask{
					SubtaskID:   int(subtask.EvaluationSubtasks.SubtaskID),
					Points:      int(subtask.EvaluationSubtasks.SubtaskPoints),
					Accepted:    int(subtask.EvaluationSubtasks.Accepted),
					Wrong:       int(subtask.EvaluationSubtasks.Wrong),
					Untested:    int(subtask.EvaluationSubtasks.Untested),
					Description: description,
				})
			}
		}

		// Process test groups
		testGroups := make([]TestGroup, 0)
		testGroupModels, hasTestGroups := submUUIDToTestGroups[subm.Submissions.SubmUUID]
		if hasTestGroups {
			for _, testGroup := range testGroupModels {
				subtaskArrayStrPtr := testGroup.EvaluationTestgroups.StatementSubtasks
				subtaskArray := []int{}
				if subtaskArrayStrPtr != nil {
					// {1,2,3}
					subtaskArrayStr := strings.Trim(*subtaskArrayStrPtr, "{}")
					subtaskArrayStrs := strings.Split(subtaskArrayStr, ",")
					for _, subtaskStr := range subtaskArrayStrs {
						subtask, err := strconv.Atoi(subtaskStr)
						if err != nil {
							log.Printf("Error converting subtask string to int: %v (took %v)", err, time.Since(startTotal))
							return nil, fmt.Errorf("failed to convert subtask string to int: %w", err)
						}
						subtaskArray = append(subtaskArray, subtask)
					}
				}
				testGroups = append(testGroups, TestGroup{
					TestGroupID: int(testGroup.EvaluationTestgroups.TestgroupID),
					Points:      int(testGroup.EvaluationTestgroups.TestgroupPoints),
					Accepted:    int(testGroup.EvaluationTestgroups.Accepted),
					Wrong:       int(testGroup.EvaluationTestgroups.Wrong),
					Untested:    int(testGroup.EvaluationTestgroups.Untested),
					Subtasks:    subtaskArray,
				})
			}
		}

		submissions[i] = &Submission{
			UUID:    subm.Submissions.SubmUUID,
			Content: subm.Submissions.Content,
			Author: Author{
				UUID:     subm.Submissions.AuthorUUID,
				Username: usernames[i],
			},
			Task: Task{
				ShortID:  subm.Submissions.TaskID,
				FullName: taskFullNames[i],
			},
			Lang: Lang{
				ShortID:  subm.Evaluations.LangID,
				Display:  langIDToFullName[subm.Evaluations.LangID],
				MonacoID: langIDToMonacoID[subm.Evaluations.LangID],
			},
			CurrEval: Evaluation{
				UUID:       subm.Evaluations.EvalUUID,
				Stage:      subm.Evaluations.EvaluationStage,
				CreatedAt:  subm.Evaluations.CreatedAt,
				Subtasks:   subtasks,
				TestGroups: testGroups,
				TestSet: &TestSet{
					Accepted: int(subm.EvaluationTestset.Accepted),
					Wrong:    int(subm.EvaluationTestset.Wrong),
					Untested: int(subm.EvaluationTestset.Untested),
				},
			},
			CreatedAt: subm.Submissions.CreatedAt,
		}
	}
	log.Printf("Processed submissions in %v", time.Since(start))

	log.Printf("Total ListSubmissions execution time: %v", time.Since(startTotal))
	return submissions, nil
}

func (s *SubmissionSrvc) GetSubmission(ctx context.Context, submUuid uuid.UUID) (*Submission, error) {
	selectStmt := postgres.SELECT(table.Submissions.AllColumns).
		FROM(table.Submissions).
		WHERE(table.Submissions.SubmUUID.EQ(postgres.UUID(submUuid)))

	var subms []model.Submissions
	if err := selectStmt.QueryContext(ctx, s.postgres, &subms); err != nil {
		format := "failed to get submission: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}
	if len(subms) == 0 {
		return nil, ErrSubmissionNotFound()
	}
	if len(subms) > 1 {
		format := "multiple submissions found with the same UUID: %s"
		errMsg := fmt.Errorf(format, submUuid)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	submRow := subms[0]

	user, err := s.userSrvc.GetUserByUUID(ctx, submRow.AuthorUUID)
	if err != nil {
		return nil, err
	}

	task, err := s.taskSrvc.GetTask(ctx, submRow.TaskID)
	if err != nil {
		return nil, err
	}

	langs, err := planglist.ListProgrammingLanguages()
	if err != nil {
		return nil, err
	}

	var lang planglist.ProgrammingLang
	found := false
	for _, l := range langs {
		if l.ID == submRow.ProgLangID {
			lang = l
			found = true
			break
		}
	}
	if !found {
		return nil, ErrInvalidProgLang()
	}

	if submRow.CurrentEvalUUID == nil {
		return nil, ErrEvaluationNotSet()
	}
	eval, err := s.getEvaluation(ctx, *submRow.CurrentEvalUUID)
	if err != nil {
		return nil, err
	}
	if eval == nil {
		format := "evaluation is nil for submission: %s"
		errMsg := fmt.Errorf(format, submUuid)
		return nil, ErrEvaluationNotSet().SetDebug(errMsg)
	}

	subm := Submission{
		UUID:    submRow.SubmUUID,
		Content: submRow.Content,
		Author: Author{
			UUID:     user.UUID,
			Username: user.Username,
		},
		Task: Task{
			ShortID:  task.ShortId,
			FullName: task.FullName,
		},
		Lang: Lang{
			ShortID:  submRow.ProgLangID,
			Display:  lang.FullName,
			MonacoID: lang.MonacoId,
		},
		CurrEval:  *eval,
		CreatedAt: submRow.CreatedAt,
	}

	return &subm, nil
}

func (s *SubmissionSrvc) getEvaluation(ctx context.Context, evalUUID uuid.UUID) (*Evaluation, error) {
	selectStmt := postgres.SELECT(table.Evaluations.AllColumns).
		FROM(table.Evaluations).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUUID)))

	var eval model.Evaluations
	if err := selectStmt.QueryContext(ctx, s.postgres, &eval); err != nil {
		format := "failed to get evaluation: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	// Get subtasks
	selectSubtasks := postgres.SELECT(table.EvaluationSubtasks.AllColumns).
		FROM(table.EvaluationSubtasks).
		WHERE(table.EvaluationSubtasks.EvalUUID.EQ(postgres.UUID(evalUUID)))

	var subtasks []model.EvaluationSubtasks
	if err := selectSubtasks.QueryContext(ctx, s.postgres, &subtasks); err != nil {
		format := "failed to get subtasks: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	// Get test groups
	selectTestGroups := postgres.SELECT(table.EvaluationTestgroups.AllColumns).
		FROM(table.EvaluationTestgroups).
		WHERE(table.EvaluationTestgroups.EvalUUID.EQ(postgres.UUID(evalUUID)))

	var testGroups []model.EvaluationTestgroups
	if err := selectTestGroups.QueryContext(ctx, s.postgres, &testGroups); err != nil {
		format := "failed to get test groups: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	// Get test set
	selectTestSet := postgres.SELECT(table.EvaluationTestset.AllColumns).
		FROM(table.EvaluationTestset).
		WHERE(table.EvaluationTestset.EvalUUID.EQ(postgres.UUID(evalUUID)))

	var testSet model.EvaluationTestset
	if err := selectTestSet.QueryContext(ctx, s.postgres, &testSet); err != nil {
		format := "failed to get test set: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	// Process subtasks
	subtasksList := make([]Subtask, 0, len(subtasks))
	for _, subtask := range subtasks {
		description := ""
		if subtask.Description != nil {
			description = *subtask.Description
		}
		subtasksList = append(subtasksList, Subtask{
			SubtaskID:   int(subtask.SubtaskID),
			Points:      int(subtask.SubtaskPoints),
			Accepted:    int(subtask.Accepted),
			Wrong:       int(subtask.Wrong),
			Untested:    int(subtask.Untested),
			Description: description,
		})
	}

	// Process test groups
	testGroupsList := make([]TestGroup, 0, len(testGroups))
	for _, testGroup := range testGroups {
		subtaskArray := []int{}
		if testGroup.StatementSubtasks != nil {
			subtaskArrayStr := strings.Trim(*testGroup.StatementSubtasks, "{}")
			subtaskStrs := strings.Split(subtaskArrayStr, ",")
			for _, subtaskStr := range subtaskStrs {
				subtask, err := strconv.Atoi(subtaskStr)
				if err != nil {
					format := "failed to convert subtask string to int: %w"
					errMsg := fmt.Errorf(format, err)
					return nil, ErrInternalSE().SetDebug(errMsg)
				}
				subtaskArray = append(subtaskArray, subtask)
			}
		}
		testGroupsList = append(testGroupsList, TestGroup{
			TestGroupID: int(testGroup.TestgroupID),
			Points:      int(testGroup.TestgroupPoints),
			Accepted:    int(testGroup.Accepted),
			Wrong:       int(testGroup.Wrong),
			Untested:    int(testGroup.Untested),
			Subtasks:    subtaskArray,
		})
	}

	return &Evaluation{
		UUID:       eval.EvalUUID,
		Stage:      eval.EvaluationStage,
		CreatedAt:  eval.CreatedAt,
		Subtasks:   subtasksList,
		TestGroups: testGroupsList,
		TestSet: &TestSet{
			Accepted: int(testSet.Accepted),
			Wrong:    int(testSet.Wrong),
			Untested: int(testSet.Untested),
		},
	}, nil
}
