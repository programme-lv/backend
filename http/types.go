package http

// Evaluation represents the evaluation of a submission.
type Evaluation struct {
	UUID          string `json:"uuid"`
	Status        string `json:"status"`
	ReceivedScore int    `json:"receivedScore"`
	PossibleScore int    `json:"possibleScore"`
}

// TestGroupResult represents the result of a test group.
type TestGroupResult struct {
	TestGroupID      int `json:"test_group_id"`
	TestGroupScore   int `json:"test_group_score"`
	StatementSubtask int `json:"statement_subtask"`
	AcceptedTests    int `json:"accepted_tests"`
	WrongTests       int `json:"wrong_tests"`
	UntestedTests    int `json:"untested_tests"`
}

// TestsResult represents the result of tests.
type TestsResult struct {
	Accepted int `json:"accepted"`
	Wrong    int `json:"wrong"`
	Untested int `json:"untested"`
}

// SubtaskResult represents the result of a subtask.
type SubtaskResult struct {
	SubtaskID     int `json:"subtask_id"`
	SubtaskScore  int `json:"subtask_score"`
	AcceptedTests int `json:"accepted_tests"`
	WrongTests    int `json:"wrong_tests"`
	UntestedTests int `json:"untested_tests"`
}

// Submission represents a code submission.
type Submission struct {
	SubmUUID              string            `json:"subm_uuid"`
	Submission            string            `json:"submission"`
	Username              string            `json:"username"`
	CreatedAt             string            `json:"created_at"`
	EvalUUID              string            `json:"eval_uuid"`
	EvalStatus            string            `json:"eval_status"`
	EvalScoringTestgroups []TestGroupResult `json:"eval_scoring_testgroups"`
	EvalScoringTests      TestsResult       `json:"eval_scoring_tests"`
	EvalScoringSubtasks   []SubtaskResult   `json:"eval_scoring_subtasks"`
	PLangID               string            `json:"p_lang_id"`
	PLangDisplayName      string            `json:"p_lang_display_name"`
	PLangMonacoID         string            `json:"p_lang_monaco_id"`
	TaskName              string            `json:"task_name"`
	TaskID                string            `json:"task_id"`
}

// SubmissionStateUpdate represents the update state of a submission.
type SubmissionStateUpdate struct {
	SubmUUID string `json:"subm_uuid"`
	EvalUUID string `json:"eval_uuid"`
	NewState string `json:"new_state"`
}

// TestGroupScoreUpdate represents the score update for a test group.
type TestGroupScoreUpdate struct {
	SubmUUID      string `json:"subm_uuid"`
	EvalUUID      string `json:"eval_uuid"`
	TestGroupID   int    `json:"test_group_id"`
	AcceptedTests int    `json:"accepted_tests"`
	WrongTests    int    `json:"wrong_tests"`
	UntestedTests int    `json:"untested_tests"`
}

// SubmissionListUpdate represents the update of a submission list.
type SubmissionListUpdate struct {
	SubmCreated        Submission            `json:"subm_created"`
	StateUpdate        SubmissionStateUpdate `json:"state_update"`
	TestGroupResUpdate TestGroupScoreUpdate  `json:"testgroup_res_update"`
}

// ProgrammingLang represents a programming language.
type ProgrammingLang struct {
	ID               string `json:"id"`
	FullName         string `json:"fullName"`
	CodeFilename     string `json:"codeFilename"`
	CompileCmd       string `json:"compileCmd"`
	ExecuteCmd       string `json:"executeCmd"`
	EnvVersionCmd    string `json:"envVersionCmd"`
	HelloWorldCode   string `json:"helloWorldCode"`
	MonacoID         string `json:"monacoId"`
	CompiledFilename string `json:"compiledFilename"`
	Enabled          bool   `json:"enabled"`
}

// Task represents a competitive programming task.
type Task struct {
	PublishedTaskID        string            `json:"published_task_id"`
	TaskFullName           string            `json:"task_full_name"`
	MemoryLimitMegabytes   int               `json:"memory_limit_megabytes"`
	CPUTimeLimitSeconds    float64           `json:"cpu_time_limit_seconds"`
	OriginOlympiad         string            `json:"origin_olympiad"`
	IllustrationImgURL     string            `json:"illustration_img_url"`
	DifficultyRating       int               `json:"difficulty_rating"`
	DefaultMDStatement     MarkdownStatement `json:"default_md_statement"`
	Examples               []Example         `json:"examples"`
	DefaultPDFStatementURL string            `json:"default_pdf_statement_url"`
	OriginNotes            map[string]string `json:"origin_notes"`
	VisibleInputSubtasks   []StInputs        `json:"visible_input_subtasks"`
}

// TaskSubmEvalData represents the evaluation data of a task submission.
type TaskSubmEvalData struct {
	PublishedTaskID      string                         `json:"published_task_id"`
	TaskFullName         string                         `json:"task_full_name"`
	MemoryLimitMegabytes int                            `json:"memory_limit_megabytes"`
	CPUTimeLimitSeconds  float64                        `json:"cpu_time_limit_seconds"`
	Tests                []TaskEvalTestInformation      `json:"tests"`
	TestlibCheckerCode   string                         `json:"testlib_checker_code"`
	SubtaskScores        []TaskEvalSubtaskScore         `json:"subtask_scores"`
	TestGroupInformation []TaskEvalTestGroupInformation `json:"test_group_information"`
}

// TaskEvalSubtaskScore represents the score of a subtask.
type TaskEvalSubtaskScore struct {
	SubtaskID int `json:"subtask_id"`
	Score     int `json:"score"`
}

// TaskEvalTestGroupInformation represents the information of a test group.
type TaskEvalTestGroupInformation struct {
	TestGroupID int `json:"test_group_id"`
	Score       int `json:"score"`
	Subtask     int `json:"subtask"`
}

// TaskEvalTestInformation represents the information of a test in task evaluation.
type TaskEvalTestInformation struct {
	TestID          int    `json:"test_id"`
	FullInputS3URI  string `json:"full_input_s3_uri"`
	InputSHA256     string `json:"input_sha256"`
	FullAnswerS3URI string `json:"full_answer_s3_uri"`
	AnswerSHA256    string `json:"answer_sha256"`
	Subtasks        []int  `json:"subtasks"`
	TestGroup       int    `json:"test_group"`
}

// MarkdownStatement represents a markdown statement for a task.
type MarkdownStatement struct {
	Story   string `json:"story"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Notes   string `json:"notes,omitempty"`
	Scoring string `json:"scoring,omitempty"`
}

// StInputs represents subtask inputs for a task.
type StInputs struct {
	Subtask int      `json:"subtask"`
	Inputs  []string `json:"inputs"`
}

// Example represents an example for a task.
type Example struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	MDNote string `json:"md_note,omitempty"`
}

// User represents a user.
type User struct {
	UUID      string `json:"uuid"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

// UserPayload represents the payload for creating and updating a user.
type UserPayload struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Password  string `json:"password"`
}

// SecureUUIDPayload defines a payload with a JWT token and UUID.
type SecureUUIDPayload struct {
	Token string `json:"token"`
	UUID  string `json:"uuid"`
}

// JwtClaims represents the claims in a JWT token.
type JwtClaims struct {
	Username  string   `json:"username"`
	Firstname string   `json:"firstname"`
	Lastname  string   `json:"lastname"`
	Email     string   `json:"email"`
	UUID      string   `json:"uuid"`
	Scopes    []string `json:"scopes"`
	Issuer    string   `json:"issuer"`
	Subject   string   `json:"subject"`
	Audience  []string `json:"audience"`
	ExpiresAt string   `json:"expires_at"`
	IssuedAt  string   `json:"issued_at"`
	NotBefore string   `json:"not_before"`
}
