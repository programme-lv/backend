package http

import (
	"time"

	"github.com/programme-lv/backend/subm"
)

type testGroupResultResponseBody struct {
	TestGroupID      int `json:"test_group_id"`
	TestGroupScore   int `json:"test_group_score"`
	StatementSubtask int `json:"statement_subtask"`
	AcceptedTests    int `json:"accepted_tests"`
	WrongTests       int `json:"wrong_tests"`
	UntestedTests    int `json:"untested_tests"`
}

type testsScoringResultResponseBody struct {
	Accepted int `json:"accepted"`
	Wrong    int `json:"wrong"`
	Untested int `json:"untested"`
}

type subtaskResultResponseBody struct {
	SubtaskID     int `json:"subtask_id"`
	SubtaskScore  int `json:"subtask_score"`
	AcceptedTests int `json:"accepted_tests"`
	WrongTests    int `json:"wrong_tests"`
	UntestedTests int `json:"untested_tests"`
}

type BriefSubmission struct {
	SubmUUID              string                          `json:"subm_uuid"`
	Username              string                          `json:"username"`
	CreatedAt             string                          `json:"created_at"`
	EvalUUID              string                          `json:"eval_uuid"`
	EvalStatus            string                          `json:"eval_status"`
	EvalScoringTestgroups []*testGroupResultResponseBody  `json:"eval_scoring_testgroups,omitempty"`
	EvalScoringTests      *testsScoringResultResponseBody `json:"eval_scoring_tests,omitempty"`
	EvalScoringSubtasks   []*subtaskResultResponseBody    `json:"eval_scoring_subtasks,omitempty"`
	PLangID               string                          `json:"p_lang_id"`
	PLangDisplayName      string                          `json:"p_lang_display_name"`
	PLangMonacoID         string                          `json:"p_lang_monaco_id"`
	TaskName              string                          `json:"task_name"`
	TaskID                string                          `json:"task_id"`
}

type EvalTestResults struct {
	TestId   int  `json:"test_id"`
	Reached  bool `json:"reached"`
	Ignored  bool `json:"ignored"`
	Finished bool `json:"finished"`

	InputTrimmed  *string `json:"input_trimmed"`
	AnswerTrimmed *string `json:"answer_trimmed"`

	TimeLimitExceeded   *bool `json:"time_limit_exceeded"`
	MemoryLimitExceeded *bool `json:"memory_limit_exceeded"`

	Subtasks  []int `json:"subtasks"`
	TestGroup *int  `json:"test_group"`

	SubmCpuTimeMillis *int    `json:"subm_cpu_time_millis"`
	SubmMemKibiBytes  *int    `json:"subm_mem_kibi_bytes"`
	SubmWallTime      *int    `json:"subm_wall_time"`
	SubmExitCode      *int    `json:"subm_exit_code"`
	SubmStdoutTrimmed *string `json:"subm_stdout_trimmed"`
	SubmStderrTrimmed *string `json:"subm_stderr_trimmed"`
	SubmExitSignal    *int    `json:"subm_exit_signal"`

	CheckerCpuTimeMillis *int    `json:"checker_cpu_time_millis"`
	CheckerMemKibiBytes  *int    `json:"checker_mem_kibi_bytes"`
	CheckerWallTime      *int    `json:"checker_wall_time"`
	CheckerExitCode      *int    `json:"checker_exit_code"`
	CheckerStdoutTrimmed *string `json:"checker_stdout_trimmed"`
	CheckerStderrTrimmed *string `json:"checker_stderr_trimmed"`
}

type EvalDetails struct {
	EvalUuid string `json:"eval_uuid"`

	CreatedAtRfc3339 string  `json:"created_at_rfc3339"`
	ErrorMsg         *string `json:"error_msg"`
	EvalStage        string  `json:"eval_stage"`

	CpuTimeLimitMillis   *int `json:"cpu_time_limit_millis"`
	MemoryLimitKibiBytes *int `json:"memory_limit_kibi_bytes"`

	ProgrammingLang   ProgrammingLang `json:"programming_lang"`
	SystemInformation *string         `json:"system_information"`

	CompileCpuTimeMillis *int    `json:"compile_cpu_time_millis"`
	CompileMemKibiBytes  *int    `json:"compile_mem_kibi_bytes"`
	CompileWallTime      *int    `json:"compile_wall_time"`
	CompileExitCode      *int    `json:"compile_exit_code"`
	CompileStdoutTrimmed *string `json:"compile_stdout_trimmed"`
	CompileStderrTrimmed *string `json:"compile_stderr_trimmed"`
}

type FullSubmission struct {
	BriefSubmission
	SubmContent     string             `json:"subm_content"`
	EvalTestResults []*EvalTestResults `json:"eval_test_results"`
	EvalDetails     *EvalDetails       `json:"eval_details"`
}

func mapEvalTestResults(x *subm.EvalTestResults) *EvalTestResults {
	if x == nil {
		return nil
	}

	return &EvalTestResults{
		TestId:               x.TestId,
		Reached:              x.Reached,
		Ignored:              x.Ignored,
		Finished:             x.Finished,
		InputTrimmed:         x.InputTrimmed,
		AnswerTrimmed:        x.AnswerTrimmed,
		TimeLimitExceeded:    x.TimeLimitExceeded,
		MemoryLimitExceeded:  x.MemoryLimitExceeded,
		Subtasks:             x.Subtasks,
		TestGroup:            x.TestGroup,
		SubmCpuTimeMillis:    x.SubmCpuTimeMillis,
		SubmMemKibiBytes:     x.SubmMemKibiBytes,
		SubmWallTime:         x.SubmWallTime,
		SubmExitCode:         x.SubmExitCode,
		SubmStdoutTrimmed:    x.SubmStdoutTrimmed,
		SubmStderrTrimmed:    x.SubmStderrTrimmed,
		CheckerCpuTimeMillis: x.CheckerCpuTimeMillis,
		CheckerMemKibiBytes:  x.CheckerMemKibiBytes,
		CheckerWallTime:      x.CheckerWallTime,
		CheckerExitCode:      x.CheckerExitCode,
		CheckerStdoutTrimmed: x.CheckerStdoutTrimmed,
		CheckerStderrTrimmed: x.CheckerStderrTrimmed,
		SubmExitSignal:       x.SubmExitSignal,
	}
}

func mapEvalTestResultsSlice(x []*subm.EvalTestResults) []*EvalTestResults {
	if x == nil {
		return nil
	}

	res := make([]*EvalTestResults, len(x))
	for i, v := range x {
		res[i] = mapEvalTestResults(v)
	}
	return res
}

func mapProgrammingLang(x subm.ProgrammingLang) ProgrammingLang {
	return ProgrammingLang{
		ID:               x.ID,
		FullName:         x.FullName,
		CodeFilename:     x.CodeFilename,
		CompileCmd:       x.CompileCmd,
		ExecuteCmd:       x.ExecuteCmd,
		EnvVersionCmd:    x.EnvVersionCmd,
		HelloWorldCode:   x.HelloWorldCode,
		MonacoID:         x.MonacoId,
		CompiledFilename: x.CompiledFilename,
		Enabled:          x.Enabled,
	}
}

func mapEvalDetails(x *subm.EvalDetails) *EvalDetails {
	return &EvalDetails{
		EvalUuid:             x.EvalUuid,
		CreatedAtRfc3339:     x.CreatedAt.UTC().Format(time.RFC3339),
		ErrorMsg:             x.ErrorMsg,
		EvalStage:            x.EvalStage,
		CpuTimeLimitMillis:   x.CpuTimeLimitMillis,
		MemoryLimitKibiBytes: x.MemoryLimitKibiBytes,
		ProgrammingLang:      mapProgrammingLang(x.ProgrammingLang),
		SystemInformation:    x.SystemInformation,

		CompileCpuTimeMillis: x.CompileCpuTimeMillis,
		CompileMemKibiBytes:  x.CompileMemKibiBytes,
		CompileWallTime:      x.CompileWallTime,
		CompileExitCode:      x.CompileExitCode,
		CompileStdoutTrimmed: x.CompileStdoutTrimmed,
		CompileStderrTrimmed: x.CompileStderrTrimmed,
	}
}

func mapFullSubm(x *subm.FullSubmission) *FullSubmission {
	res := &FullSubmission{
		BriefSubmission: *mapBriefSubm(&x.BriefSubmission),
		SubmContent:     x.SubmContent,
		EvalTestResults: mapEvalTestResultsSlice(x.EvalTestResults),
		EvalDetails:     mapEvalDetails(x.EvalDetails),
	}

	return res
}
func mapBriefSubm(x *subm.BriefSubmission) *BriefSubmission {
	if x == nil {
		return nil
	}

	mapEvalTestGroupResults := func(testGroups []*subm.TestGroupResult) []*testGroupResultResponseBody {
		if testGroups == nil {
			return nil
		}
		result := make([]*testGroupResultResponseBody, len(testGroups))
		for i, tg := range testGroups {
			result[i] = &testGroupResultResponseBody{
				TestGroupID:      tg.TestGroupID,
				TestGroupScore:   tg.TestGroupScore,
				StatementSubtask: tg.StatementSubtask,
				AcceptedTests:    tg.AcceptedTests,
				WrongTests:       tg.WrongTests,
				UntestedTests:    tg.UntestedTests,
			}
		}
		return result
	}

	mapEvalTestsResult := func(tests *subm.TestsResult) *testsScoringResultResponseBody {
		if tests == nil {
			return nil
		}
		return &testsScoringResultResponseBody{
			Accepted: tests.Accepted,
			Wrong:    tests.Wrong,
			Untested: tests.Untested,
		}
	}

	mapEvalSubtaskResults := func(subtasks []*subm.SubtaskResult) []*subtaskResultResponseBody {
		if subtasks == nil {
			return nil
		}
		result := make([]*subtaskResultResponseBody, len(subtasks))
		for i, st := range subtasks {
			result[i] = &subtaskResultResponseBody{
				SubtaskID:     st.SubtaskID,
				SubtaskScore:  st.SubtaskScore,
				AcceptedTests: st.AcceptedTests,
				WrongTests:    st.WrongTests,
				UntestedTests: st.UntestedTests,
			}
		}
		return result
	}

	return &BriefSubmission{
		SubmUUID:              x.SubmUUID,
		Username:              x.Username,
		CreatedAt:             x.CreatedAt,
		EvalUUID:              x.EvalUUID,
		EvalStatus:            x.EvalStatus,
		EvalScoringTestgroups: mapEvalTestGroupResults(x.EvalScoringTestgroups),
		EvalScoringTests:      mapEvalTestsResult(x.EvalScoringTests),
		EvalScoringSubtasks:   mapEvalSubtaskResults(x.EvalScoringSubtasks),
		PLangID:               x.PLangID,
		PLangDisplayName:      x.PLangDisplayName,
		PLangMonacoID:         x.PLangMonacoID,
		TaskName:              x.TaskName,
		TaskID:                x.TaskID,
	}
}
