package http

import (
	"time"

	"github.com/programme-lv/backend/submsrvc"
)

type TestGroup struct {
	TestGroupID    int   `json:"test_group_id"`
	TestGroupScore int   `json:"test_group_score"`
	AcceptedTests  int   `json:"accepted_tests"`
	WrongTests     int   `json:"wrong_tests"`
	UntestedTests  int   `json:"untested_tests"`
	Subtasks       []int `json:"subtasks"`
}

type TestSet struct {
	Accepted int `json:"accepted"`
	Wrong    int `json:"wrong"`
	Untested int `json:"untested"`
}

type Subtask struct {
	SubtaskID     int    `json:"subtask_id"`
	SubtaskScore  int    `json:"subtask_score"`
	AcceptedTests int    `json:"accepted_tests"`
	WrongTests    int    `json:"wrong_tests"`
	UntestedTests int    `json:"untested_tests"`
	Description   string `json:"description"`
}

type BriefSubmission struct {
	SubmUUID          string      `json:"subm_uuid"`
	Username          string      `json:"username"`
	CreatedAt         string      `json:"created_at"`
	EvalUUID          string      `json:"eval_uuid"`
	EvalStatus        string      `json:"eval_status"`
	TestGroups        []TestGroup `json:"test_groups"`
	TestSet           *TestSet    `json:"test_set"`
	Subtasks          []Subtask   `json:"subtasks"`
	ProgrLangID       string      `json:"p_lang_id"`
	ProgrLangName     string      `json:"p_lang_display_name"`
	ProgrLangMonacoID string      `json:"p_lang_monaco_id"`
	TaskFullName      string      `json:"task_name"`
	TaskShortID       string      `json:"task_id"`
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
	ExitSignal    *int64  `json:"exit_signal"`
}

type EvalTest struct {
	TestId int `json:"test_id"`

	Reached  bool `json:"reached"`
	Ignored  bool `json:"ignored"`
	Finished bool `json:"finished"`

	InputTrimmed  *string `json:"input_trimmed"`
	AnswerTrimmed *string `json:"answer_trimmed"`

	TimeExceeded   bool `json:"time_exceeded"`
	MemoryExceeded bool `json:"memory_exceeded"`

	Subtasks  []int `json:"subtasks"`
	TestGroup []int `json:"test_groups"`

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
		TestGroups:        mapTestGroupScoring(x.CurrEval.Groups),
		TestSet:           mapTestsScore(x.CurrEval.TestSet),
		Subtasks:          mapSubtasksScore(x.CurrEval.Subtasks),
		ProgrLangID:       x.Lang.ShortID,
		ProgrLangName:     x.Lang.Display,
		ProgrLangMonacoID: x.Lang.MonacoID,
		TaskFullName:      x.Task.FullName,
		TaskShortID:       x.Task.ShortID,
	}
}

func mapTestGroupScoring(x []submsrvc.TestGroup) []TestGroup {
	if x == nil {
		return nil
	}
	res := make([]TestGroup, len(x))
	for i, v := range x {
		res[i] = TestGroup{
			TestGroupID:    v.GroupID,
			TestGroupScore: v.Points,
			AcceptedTests:  v.Accepted,
			WrongTests:     v.Wrong,
			UntestedTests:  v.Untested,
			Subtasks:       v.Subtasks,
		}
	}
	return res
}

func mapTestsScore(x *submsrvc.TestSet) *TestSet {
	if x == nil {
		return nil
	}
	return &TestSet{
		Accepted: x.Accepted,
		Wrong:    x.Wrong,
		Untested: x.Untested,
	}
}

func mapSubtasksScore(x []submsrvc.Subtask) []Subtask {
	if x == nil {
		return nil
	}
	res := make([]Subtask, len(x))
	for i, v := range x {
		res[i] = Subtask{
			SubtaskID:     v.SubtaskID,
			SubtaskScore:  v.Points,
			AcceptedTests: v.Accepted,
			WrongTests:    v.Wrong,
			UntestedTests: v.Untested,
			Description:   v.Description,
		}
	}
	return res
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
	return &EvalDetails{
		EvalUuid:             x.EvalUuid,
		CreatedAtRfc3339:     x.CreatedAt.Format(time.RFC3339),
		ErrorMsg:             x.ErrorMsg,
		EvalStage:            x.EvalStage,
		CpuTimeLimitMillis:   &x.CpuTimeLimitMillis,
		MemoryLimitKibiBytes: &x.MemoryLimitKiB,
		ProgrLang: ProgrammingLang{
			ID:               x.ProgrammingLang.ID,
			FullName:         x.ProgrammingLang.FullName,
			CodeFilename:     x.ProgrammingLang.CodeFilename,
			CompileCmd:       x.ProgrammingLang.CompileCmd,
			ExecuteCmd:       x.ProgrammingLang.ExecuteCmd,
			EnvVersionCmd:    x.ProgrammingLang.EnvVersionCmd,
			HelloWorldCode:   x.ProgrammingLang.HelloWorldCode,
			MonacoID:         x.ProgrammingLang.MonacoId,
			CompiledFilename: x.ProgrammingLang.CompiledFilename,
			Enabled:          x.ProgrammingLang.Enabled,
		},
		SystemInfo:      x.SystemInformation,
		CompileExecInfo: mapExecInfo(x.CompileRuntime),
	}
}

func mapExecInfo(x *submsrvc.RuntimeData) *ExecutionInfo {
	if x == nil {
		return nil
	}
	return &ExecutionInfo{
		CpuTimeMillis: x.CpuMillis,
		MemKibiBytes:  x.MemoryKiB,
		WallTime:      x.WallTime,
		ExitCode:      x.ExitCode,
		StdoutTrimmed: x.Stdout,
		StderrTrimmed: x.Stderr,
		ExitSignal:    x.ExitSignal,
	}
}

func mapTestResults(x []submsrvc.EvalTestResult) []EvalTest {
	if x == nil {
		return nil
	}
	res := make([]EvalTest, len(x))
	for i, v := range x {
		res[i] = mapEvalTest(v)
	}
	return res
}

func mapEvalTest(x submsrvc.EvalTestResult) EvalTest {
	return EvalTest{
		TestId:          x.TestId,
		Reached:         x.Reached,
		Ignored:         x.Ignored,
		Finished:        x.Finished,
		InputTrimmed:    x.InputTrimmed,
		AnswerTrimmed:   x.AnswerTrimmed,
		TimeExceeded:    x.TimeExceeded,
		MemoryExceeded:  x.MemoryExceeded,
		Subtasks:        x.Subtasks,
		TestGroup:       x.TestGroups,
		SubmExecInfo:    mapExecInfo(x.SubmRuntime),
		CheckerExecInfo: mapExecInfo(x.CheckerRuntime),
	}
}
