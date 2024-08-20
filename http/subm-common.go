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

type testsResultResponseBody struct {
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
	SubmUUID              string                         `json:"subm_uuid"`
	Username              string                         `json:"username"`
	CreatedAt             string                         `json:"created_at"`
	EvalUUID              string                         `json:"eval_uuid"`
	EvalStatus            string                         `json:"eval_status"`
	EvalScoringTestgroups []*testGroupResultResponseBody `json:"eval_scoring_testgroups,omitempty"`
	EvalScoringTests      *testsResultResponseBody       `json:"eval_scoring_tests,omitempty"`
	EvalScoringSubtasks   []*subtaskResultResponseBody   `json:"eval_scoring_subtasks,omitempty"`
	PLangID               string                         `json:"p_lang_id"`
	PLangDisplayName      string                         `json:"p_lang_display_name"`
	PLangMonacoID         string                         `json:"p_lang_monaco_id"`
	TaskName              string                         `json:"task_name"`
	TaskID                string                         `json:"task_id"`
}

func mapSubm(x *subm.BriefSubmission) *BriefSubmission {
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

	mapEvalTestsResult := func(tests *subm.TestsResult) *testsResultResponseBody {
		if tests == nil {
			return nil
		}
		return &testsResultResponseBody{
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
