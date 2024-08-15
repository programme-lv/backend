package subm

type SubmissionDetailsRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // subm#details

	Content string `dynamodbav:"subm_content"` // submission task solution code

	AuthorUuid string `dynamodbav:"author_uuid"`
	TaskUuid   string `dynamodbav:"task_uuid"`
	ProgLangId string `dynamodbav:"prog_lang_id"`

	CurrentEvalUuid   string `dynamodbav:"current_eval_uuid"`   // the uuid of the current evaluation
	CurrentEvalStatus string `dynamodbav:"current_eval_status"` // "waiting", "received", "compiling", "testing", "finished"

	Gsi1Pk int `dynamodbav:"gsi1_pk"` // gsi1pk = 1

	CreatedAtRfc3339 string `dynamodbav:"created_at_rfc3339_utc"`
	Version          int64  `dynamodbav:"version"` // For optimistic locking
}

type SubmissionScoringTestsRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // subm#scoring#tests

	Accepted int `dynamodbav:"accepted"`
	Wrong    int `dynamodbav:"wrong"`
	Untested int `dynamodbav:"untested"`

	Gsi1Pk int `dynamodbav:"gsi1_pk"` // gsi1pk = 1
}

type SubmissionScoringSubtaskRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // subm#scoring#subtask#<subtask_id>

	ReceivedScore int `dynamodbav:"received_score"`
	PossibleScore int `dynamodbav:"possible_score"`

	AcceptedTests int `dynamodbav:"accepted_tests"`
	WrongTests    int `dynamodbav:"wrong_tests"`
	UntestedTests int `dynamodbav:"untested_tests"`

	Gsi1Pk int `dynamodbav:"gsi1_pk"` // gsi1pk = 1
}

type SubmissionScoringTestgroupRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // subm#scoring#testgroup#<testgroup_id>

	StatementSubtask int `dynamodbav:"statement_subtask"`

	TestgroupScore int `dynamodbav:"testgroup_score"`

	AcceptedTests int `dynamodbav:"accepted_tests"`
	WrongTests    int `dynamodbav:"wrong_tests"`
	UntestedTests int `dynamodbav:"untested_tests"`

	Gsi1Pk int `dynamodbav:"gsi1_pk"` // gsi1pk = 1
}

type SubmissionEvaluationDetailsRow struct {
	SubmUuid string `dynamodbav:"subm_uuid"` // partition key
	SortKey  string `dynamodbav:"sort_key"`  // eval#<eval_uuid>#details

	EvalUuid        string `dynamodbav:"eval_uuid"`        // the uuid of the evaluation
	EvaluationStage string `dynamodbav:"evaluation_stage"` // "waiting", "received", "compiling", "testing", "finished"

	TestlibCheckerCode string `dynamodbav:"testlib_checker_code"` // the code of the testlib checker

	SystemInformation *string `dynamodbav:"system_information"` // details about the system that ran the evaluation

	SubmCompileStdout   *string `dynamodbav:"subm_comp_stdout"` // might be trimmed
	SubmCompileStderr   *string `dynamodbav:"subm_comp_stderr"` // might be trimmed
	SubmCompileExitCode *int64  `dynamodbav:"subm_comp_exit_code"`

	SubmCompileCpuTimeMillis   *int64 `dynamodbav:"subm_comp_cpu_time_millis"`
	SubmCompileWallTimeMillis  *int64 `dynamodbav:"subm_comp_wall_time_millis"`
	SubmCompileMemoryKibiBytes *int64 `dynamodbav:"subm_comp_memory_kibi_bytes"`

	ProgrammingLang SubmEvalDetailsProgrammingLang `dynamodbav:"programming_lang"`

	CreatedAtRfc3339 string `dynamodbav:"created_at_rfc3339_utc"`
	Version          int64  `dynamodbav:"version"` // For optimistic locking
}

type SubmissionEvaluationTestRow struct {
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
	CheckerExitCode *int64  `dynamodbav:"checker_exit_code"`

	CheckerCpuTimeMillis   *int64 `dynamodbav:"checker_cpu_time_millis"`
	CheckerWallTimeMillis  *int64 `dynamodbav:"checker_wall_time_millis"`
	CheckerMemoryKibiBytes *int64 `dynamodbav:"checker_memory_kibi_bytes"`

	SubmStdout   *string `dynamodbav:"subm_stdout"` // might be trimmed
	SubmStderr   *string `dynamodbav:"subm_stderr"` // might be trimmed
	SubmExitCode *int64  `dynamodbav:"subm_exit_code"`

	SubmCpuTimeMillis   *int64 `dynamodbav:"subm_cpu_time_millis"`
	SubmWallTimeMillis  *int64 `dynamodbav:"subm_wall_time_millis"`
	SubmMemoryKibiBytes *int64 `dynamodbav:"subm_memory_kibi_bytes"`

	Subtasks  []int `dynamodbav:"subtasks"`   // subtasks that the test is part of
	TestGroup *int  `dynamodbav:"test_group"` // test group that the test is part of
}

type SubmEvalDetailsProgrammingLang struct {
	PLangId        string  `dynamodbav:"p_lang_id"`
	DisplayName    string  `dynamodbav:"display_name"`
	SubmCodeFname  string  `dynamodbav:"subm_code_fname"`
	CompileCommand *string `dynamodbav:"compile_command"`
	CompiledFname  *string `dynamodbav:"compiled_fname"`
	ExecCommand    string  `dynamodbav:"exec_command"`
}
