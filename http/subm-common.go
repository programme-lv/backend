package http

import (
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

	InputTrimmed  string `json:"input_trimmed"`
	AnswerTrimmed string `json:"answer_trimmed"`

	TimeLimitExceeded   bool `json:"time_limit_exceeded"`
	MemoryLimitExceeded bool `json:"memory_limit_exceeded"`

	Subtasks  []int `json:"subtasks"`
	TestGroup int   `json:"test_group"`

	SubmCpuTimeMillis *int    `json:"subm_cpu_time_millis"`
	SubmMemKibiBytes  *int    `json:"subm_mem_kibi_bytes"`
	SubmWallTime      *int    `json:"subm_wall_time"`
	SubmExitCode      *int    `json:"subm_exit_code"`
	SubmStdoutTrimmed *string `json:"subm_stdout_trimmed"`
	SubmStderrTrimmed *string `json:"subm_stderr_trimmed"`

	CheckerCpuTimeMillis *int    `json:"checker_cpu_time_millis"`
	CheckerMemKibiBytes  *int    `json:"checker_mem_kibi_bytes"`
	CheckerWallTime      *int    `json:"checker_wall_time"`
	CheckerExitCode      *int    `json:"checker_exit_code"`
	CheckerStdoutTrimmed *string `json:"checker_stdout_trimmed"`
	CheckerStderrTrimmed *string `json:"checker_stderr_trimmed"`
}

type FullSubmission struct {
	BriefSubmission
	SubmContent            string             `json:"subm_content"`
	CurrentEvalTestResults []*EvalTestResults `json:"current_eval_test_results"`
}

func mapFullSubm(x *subm.FullSubmission) *FullSubmission {
	res := &FullSubmission{
		BriefSubmission:        *mapBriefSubm(&x.BriefSubmission),
		SubmContent:            x.SubmContent,
		CurrentEvalTestResults: []*EvalTestResults{},
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
