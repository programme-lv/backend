package http

import (
	"time"

	"github.com/programme-lv/backend/submsrvc"
)

type TestGroupScore struct {
	TestGroupID      int `json:"test_group_id"`
	TestGroupScore   int `json:"test_group_score"`
	StatementSubtask int `json:"statement_subtask"`
	AcceptedTests    int `json:"accepted_tests"`
	WrongTests       int `json:"wrong_tests"`
	UntestedTests    int `json:"untested_tests"`
}

type TestsScore struct {
	Accepted int `json:"accepted"`
	Wrong    int `json:"wrong"`
	Untested int `json:"untested"`
}

type SubtaskScore struct {
	SubtaskID     int `json:"subtask_id"`
	SubtaskScore  int `json:"subtask_score"`
	AcceptedTests int `json:"accepted_tests"`
	WrongTests    int `json:"wrong_tests"`
	UntestedTests int `json:"untested_tests"`
}

type BriefSubmission struct {
	SubmUUID          string           `json:"subm_uuid"`
	Username          string           `json:"username"`
	CreatedAt         string           `json:"created_at"`
	EvalUUID          string           `json:"eval_uuid"`
	EvalStatus        string           `json:"eval_status"`
	TestGroupScoring  []TestGroupScore `json:"test_group_scoring"`
	TestsScore        *TestsScore      `json:"tests_score"`
	SubtasksScore     []SubtaskScore   `json:"subtasks_score"`
	ProgrLangID       string           `json:"p_lang_id"`
	ProgrLangName     string           `json:"p_lang_display_name"`
	ProgrLangMonacoID string           `json:"p_lang_monaco_id"`
	TaskFullName      string           `json:"task_name"`
	TaskShortID       string           `json:"task_id"`
}

type FullSubmission struct {
	BriefSubmission
	Content     string       `json:"subm_content"`
	EvalDetails *EvalDetails `json:"eval_details"`
	TestResults []EvalTest   `json:"test_results"`
}

type ExecutionInfo struct {
	CpuTimeMillis int     `json:"cpu_time_millis"`
	MemKibiBytes  int     `json:"mem_kibi_bytes"`
	WallTime      int     `json:"wall_time"`
	ExitCode      int     `json:"exit_code"`
	StdoutTrimmed *string `json:"stdout_trimmed"`
	StderrTrimmed *string `json:"stderr_trimmed"`
	ExitSignal    *int    `json:"exit_signal"`
}

type EvalTest struct {
	TestId int `json:"test_id"`

	Reached  bool `json:"reached"`
	Ignored  bool `json:"ignored"`
	Finished bool `json:"finished"`

	InputTrimmed  *string `json:"input_trimmed"`
	AnswerTrimmed *string `json:"answer_trimmed"`

	TimeExceeded   *bool `json:"time_exceeded"`
	MemoryExceeded *bool `json:"memory_exceeded"`

	Subtasks  []int `json:"subtasks"`
	TestGroup *int  `json:"test_group"`

	SubmExecInfo    *ExecutionInfo `json:"subm_exec_info"`
	CheckerExecInfo *ExecutionInfo `json:"checker_exec_info"`
}

type EvalDetails struct {
	EvalUuid string `json:"eval_uuid"`

	CreatedAtRfc3339 string  `json:"created_at_rfc3339"`
	ErrorMsg         *string `json:"error_msg"`
	EvalStage        string  `json:"eval_stage"`

	CpuTimeLimitMillis   *int `json:"cpu_time_limit_millis"`
	MemoryLimitKibiBytes *int `json:"memory_limit_kibi_bytes"`

	ProgrLang  ProgrammingLang `json:"programming_lang"`
	SystemInfo *string         `json:"system_information"`

	CompileExecInfo *ExecutionInfo `json:"compile_exec_info"`
}

/*

type Submission struct {
	UUID uuid.UUID

	Content string

	Author Author
	Task   Task
	Lang   Lang

	CurrEval Evaluation

	CreatedAt time.Time
}

type Evaluation struct {
	UUID      uuid.UUID
	Stage     string
	CreatedAt time.Time

	ScoreBySubtasks   *SubtaskScoringRes
	ScoreByTestGroups *TestGroupScoringRes
	ScoreByTestSets   *TestSetScoringRes
}

type Author struct {
	UUID     uuid.UUID
	Username string
}

type Lang struct {
	ShortID  string
	Display  string
	MonacoID string
}

type Task struct {
	ShortID  string
	FullName string
}
*/

func mapBriefSubm(x *submsrvc.Submission) *BriefSubmission {
	if x == nil {
		return nil
	}
	return &BriefSubmission{
		SubmUUID:          x.UUID.String(),
		Username:          x.Author.Username,
		CreatedAt:         x.CreatedAt.Format(time.RFC3339),
		EvalUUID:          x.CurrEval.UUID.String(),
		EvalStatus:        x.CurrEval.Stage,
		TestGroupScoring:  mapTestGroupScoring(x.CurrEval.ScoreByTestGroups),
		TestsScore:        mapTestsScore(x.CurrEval.ScoreByTestSets),
		SubtasksScore:     mapSubtasksScore(x.CurrEval.ScoreBySubtasks),
		ProgrLangID:       x.Lang.ShortID,
		ProgrLangName:     x.Lang.Display,
		ProgrLangMonacoID: x.Lang.MonacoID,
		TaskFullName:      x.Task.FullName,
		TaskShortID:       x.Task.ShortID,
	}
}

func mapTestGroupScoring(x *submsrvc.TestGroupScoringRes) []TestGroupScore {
	if x == nil {
		return nil
	}
	// TODO: implement
	return []TestGroupScore{}
}

func mapTestsScore(x *submsrvc.TestSetScoringRes) *TestsScore {
	if x == nil {
		return nil
	}
	// TODO: implement
	return &TestsScore{}
}

func mapSubtasksScore(x *submsrvc.SubtaskScoringRes) []SubtaskScore {
	if x == nil {
		return nil
	}
	// TODO: implement
	return []SubtaskScore{}
}

func mapFullSubm(x *submsrvc.FullSubmission) *FullSubmission {
	if x == nil {
		return nil
	}
	return &FullSubmission{
		BriefSubmission: *mapBriefSubm(&x.Submission),
		Content:         x.Content,
		EvalDetails:     mapEvalDetails(x.EvalDetails),
		TestResults:     mapTestResults(x.TestResults),
	}
}

func mapEvalDetails(x *submsrvc.EvalDetails) *EvalDetails {
	if x == nil {
		return nil
	}
	// TODO: implement
	return &EvalDetails{
		EvalUuid: x.EvalUuid,
	}
}

func mapTestResults(x []submsrvc.EvalTest) []EvalTest {
	if x == nil {
		return nil
	}
	// TODO: implement
	res := make([]EvalTest, len(x))
	for i, v := range x {
		res[i] = EvalTest{
			TestId: v.TestId,
		}
	}
	return res
}
