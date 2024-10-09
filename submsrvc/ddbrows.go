package submsrvc

import (
	"strconv"
	"strings"
)

type SubmDetailsRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // subm#details

	Content string `dynamodbav:"subm_content"` // submission task solution code

	AuthorUuid string `dynamodbav:"author_uuid"`
	TaskId     string `dynamodbav:"task_id"`
	ProgLangId string `dynamodbav:"prog_lang_id"`

	CurrentEvalUuid   string `dynamodbav:"current_eval_uuid"`   // the uuid of the current evaluation
	CurrentEvalStatus string `dynamodbav:"current_eval_status"` // "waiting", "received", "compiling", "testing", "finished", "error"

	ErrorMsg *string `dynamodbav:"error_msg"` // error message if evaluation failed

	Gsi1Pk      int    `dynamodbav:"gsi1_pk"` // gsi1pk = 1
	Gsi1SortKey string `dynamodbav:"gsi1_sk"` // <created_at_rfc3339_utc>#<subm_uuid>#details

	CreatedAtRfc3339 string `dynamodbav:"created_at_rfc3339_utc"`
	Version          int64  `dynamodbav:"version"` // For optimistic locking
}

type SubmScoringTestsRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // subm#scoring#tests

	CurrentEvalUuid string `dynamodbav:"current_eval_uuid"` // the uuid of the current evaluation

	Accepted int `dynamodbav:"accepted_tests"`
	Wrong    int `dynamodbav:"wrong_tests"`
	Untested int `dynamodbav:"untested_tests"`

	Gsi1Pk      int    `dynamodbav:"gsi1_pk"` // gsi1pk = 1
	Gsi1SortKey string `dynamodbav:"gsi1_sk"` // <created_at_rfc3339_utc>#<subm_uuid>#scoring#tests

	// For optimistic locking, equal to current evaluation's scoring tests version
	// eval scoring row gets copied here and conditionally updated
	Version int64 `dynamodbav:"version"`
}

type SubmScoringSubtaskRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // subm#scoring#subtask#<subtask_id>

	CurrentEvalUuid string `dynamodbav:"current_eval_uuid"` // the uuid of the current evaluation

	SubtaskScore int `dynamodbav:"subtask_score"`

	AcceptedTests int `dynamodbav:"accepted_tests"`
	WrongTests    int `dynamodbav:"wrong_tests"`
	UntestedTests int `dynamodbav:"untested_tests"`

	Gsi1Pk      int    `dynamodbav:"gsi1_pk"` // gsi1pk = 1
	Gsi1SortKey string `dynamodbav:"gsi1_sk"` // <created_at_rfc3339_utc>#<subm_uuid>#scoring#subtask#<subtask_id>
	Version     int64  `dynamodbav:"version"` // For optimistic locking, equal to current evaluation's scoring subtask version
}

type SubmScoringTestgroupRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // subm#scoring#testgroup#<testgroup_id>

	CurrentEvalUuid string `dynamodbav:"current_eval_uuid"` // the uuid of the current evaluation

	StatementSubtask int `dynamodbav:"statement_subtask"`

	TestgroupScore int `dynamodbav:"testgroup_score"`

	AcceptedTests int `dynamodbav:"accepted_tests"`
	WrongTests    int `dynamodbav:"wrong_tests"`
	UntestedTests int `dynamodbav:"untested_tests"`

	Gsi1Pk      int    `dynamodbav:"gsi1_pk"` // gsi1pk = 1
	Gsi1SortKey string `dynamodbav:"gsi1_sk"` // <created_at_rfc3339_utc>#<subm_uuid>#scoring#testgroup#<testgroup_id>
	Version     int64  `dynamodbav:"version"` // For optimistic locking, equal to current evaluation's scoring testgroup version
}

func (sstgr *SubmScoringTestgroupRow) TestGroupID() int {
	// split the sort key by '#', the last element is the testgroup id
	parts := strings.Split(sstgr.SortKey, "#")
	testgroupID := parts[len(parts)-1]
	// convert the string to int
	res, err := strconv.Atoi(testgroupID)
	if err != nil {
		panic(err)
	}
	return res
}

type EvalDetailsRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // eval#<eval_uuid>#details

	EvalUuid        string `dynamodbav:"eval_uuid"`        // the uuid of the evaluation
	EvaluationStage string `dynamodbav:"evaluation_stage"` // "waiting", "received", "compiling", "testing", "finished", "error"

	CpuTimeLimitMillis *int `dynamodbav:"cpu_time_limit_millis"` // CPU time limit in milliseconds
	MemLimitKibiBytes  *int `dynamodbav:"mem_limit_kibi_bytes"`  // memory limit in kibibytes

	ErrorMsg *string `dynamodbav:"error_msg"` // error message if evaluation failed

	TestlibCheckerCode string `dynamodbav:"testlib_checker_code"` // the code of the testlib checker

	SystemInformation *string `dynamodbav:"system_information"` // details about the system that ran the evaluation

	SubmCompileStdout   *string `dynamodbav:"subm_comp_stdout"` // might be trimmed
	SubmCompileStderr   *string `dynamodbav:"subm_comp_stderr"` // might be trimmed
	SubmCompileExitCode *int    `dynamodbav:"subm_comp_exit_code"`

	SubmCompileCpuTimeMillis   *int `dynamodbav:"subm_comp_cpu_time_millis"`
	SubmCompileWallTimeMillis  *int `dynamodbav:"subm_comp_wall_time_millis"`
	SubmCompileMemoryKibiBytes *int `dynamodbav:"subm_comp_memory_kibi_bytes"`

	SubmCompileContextSwitchesForced *int64  `dynamodbav:"subm_comp_context_switches_forced"`
	SubmCompileExitSignal            *int64  `dynamodbav:"subm_comp_exit_signal"`
	SubmCompileIsolateStatus         *string `dynamodbav:"subm_comp_isolate_status"`

	ProgrammingLang EvalDetailsProgrammingLang `dynamodbav:"programming_lang"`

	CreatedAtRfc3339 string `dynamodbav:"created_at_rfc3339_utc"`
	Version          int64  `dynamodbav:"version"` // For optimistic locking
}

type EvalDetailsProgrammingLang struct {
	PLangId        string  `dynamodbav:"p_lang_id"`
	DisplayName    string  `dynamodbav:"display_name"`
	SubmCodeFname  string  `dynamodbav:"subm_code_fname"`
	CompileCommand *string `dynamodbav:"compile_command"`
	CompiledFname  *string `dynamodbav:"compiled_fname"`
	ExecCommand    string  `dynamodbav:"exec_command"`
}

type EvalTestRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // eval#<eval_uuid>#test#<padded_test_id>

	FullInputS3Uri  string `dynamodbav:"full_input_s3_uri"`
	FullAnswerS3Uri string `dynamodbav:"full_answer_s3_uri"`

	Reached  bool `dynamodbav:"reached"`
	Ignored  bool `dynamodbav:"ignored"`  // if doesn't count towards the score
	Finished bool `dynamodbav:"finished"` // if the test is evaluated / tested

	InputTrimmed  *string `dynamodbav:"input_trimmed"`  // trimmed input for display
	AnswerTrimmed *string `dynamodbav:"answer_trimmed"` // trimmed answer for display

	CheckerStdout   *string `dynamodbav:"checker_stdout"` // might be trimmed
	CheckerStderr   *string `dynamodbav:"checker_stderr"` // might be trimmed
	CheckerExitCode *int    `dynamodbav:"checker_exit_code"`

	CheckerCpuTimeMillis   *int `dynamodbav:"checker_cpu_time_millis"`
	CheckerWallTimeMillis  *int `dynamodbav:"checker_wall_time_millis"`
	CheckerMemoryKibiBytes *int `dynamodbav:"checker_memory_kibi_bytes"`

	CheckerContextSwitchesForced *int64  `dynamodbav:"checker_context_switches_forced"`
	CheckerExitSignal            *int64  `dynamodbav:"checker_exit_signal"`
	CheckerIsolateStatus         *string `dynamodbav:"checker_isolate_status"`

	SubmStdout   *string `dynamodbav:"subm_stdout"` // might be trimmed
	SubmStderr   *string `dynamodbav:"subm_stderr"` // might be trimmed
	SubmExitCode *int    `dynamodbav:"subm_exit_code"`

	SubmCpuTimeMillis   *int `dynamodbav:"subm_cpu_time_millis"`
	SubmWallTimeMillis  *int `dynamodbav:"subm_wall_time_millis"`
	SubmMemoryKibiBytes *int `dynamodbav:"subm_memory_kibi_bytes"`

	SubmContextSwitchesForced *int64  `dynamodbav:"subm_context_switches_forced"`
	SubmExitSignal            *int    `dynamodbav:"subm_exit_signal"`
	SubmIsolateStatus         *string `dynamodbav:"subm_isolate_status"`

	Subtasks  []int `dynamodbav:"subtasks"`   // subtasks that the test is part of
	TestGroup *int  `dynamodbav:"test_group"` // test group that the test is part of
}

type EvalScoringTestsRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // eval#<eval_uuid>#scoring#tests

	Accepted int `dynamodbav:"accepted_tests"`
	Wrong    int `dynamodbav:"wrong_tests"`
	Untested int `dynamodbav:"untested_tests"`

	Version int64 `dynamodbav:"version"` // For optimistic locking
}

type EvalScoringSubtaskRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // eval#<eval_uuid>#scoring#subtask#<subtask_id>

	SubtaskScore int `dynamodbav:"subtask_score"`

	AcceptedTests int `dynamodbav:"accepted_tests"`
	WrongTests    int `dynamodbav:"wrong_tests"`
	UntestedTests int `dynamodbav:"untested_tests"`

	Version int64 `dynamodbav:"version"` // For optimistic locking, equal to current evaluation's scoring subtask version
}

type EvalScoringTestgroupRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // eval#<eval_uuid>#scoring#testgroup#<testgroup_id>

	StatementSubtask int `dynamodbav:"statement_subtask"`

	TestgroupScore int `dynamodbav:"testgroup_score"`

	AcceptedTests int `dynamodbav:"accepted_tests"`
	WrongTests    int `dynamodbav:"wrong_tests"`
	UntestedTests int `dynamodbav:"untested_tests"`

	Version int64 `dynamodbav:"version"` // For optimistic locking, equal to current evaluation's scoring testgroup version
}
