package submsrvc

import (
	"time"
)

type EvalRequest struct {
	EvalUuid  string `json:"eval_uuid"`
	ResSqsUrl string `json:"res_sqs_url"`

	Code     string    `json:"code"`
	Language Language  `json:"language"`
	Tests    []ReqTest `json:"tests"`
	Checker  string    `json:"checker"`

	CpuMillis int `json:"cpu_millis"`
	MemoryKiB int `json:"memory_kib"`
}

type Language struct {
	LangID        string  `json:"lang_id"`
	LangName      string  `json:"lang_name"`
	CodeFname     string  `json:"code_fname"`
	CompileCmd    *string `json:"compile_cmd"`
	CompiledFname *string `json:"compiled_fname"`
	ExecCmd       string  `json:"exec_cmd"`
}

type ReqTest struct {
	ID int `json:"id"`

	InputSha256  string  `json:"input_sha256"`
	InputS3Url   *string `json:"input_s3_url"`
	InputContent *string `json:"input_content"`
	InputHttpUrl *string `json:"input_http_url"`

	AnswerSha256  string  `json:"answer_sha256"`
	AnswerS3Url   *string `json:"answer_s3_url"`
	AnswerContent *string `json:"answer_content"`
	AnswerHttpUrl *string `json:"answer_http_url"`
}

type EvalTestResult struct {
	TestId   int
	Reached  bool
	Ignored  bool
	Finished bool

	InputTrimmed  *string
	AnswerTrimmed *string

	TimeExceeded   bool
	MemoryExceeded bool

	Subtasks   []int
	TestGroups []int

	SubmRuntime    *RuntimeData
	CheckerRuntime *RuntimeData
}

type RuntimeData struct {
	CpuMillis  int
	MemoryKiB  int
	WallTime   int
	ExitCode   int
	Stdout     *string
	Stderr     *string
	ExitSignal *int64
}

type EvalDetails struct {
	EvalUuid string

	CreatedAt time.Time
	ErrorMsg  *string
	EvalStage string

	CpuTimeLimitMillis int
	MemoryLimitKiB     int

	ProgrammingLang   ProgrammingLang
	SystemInformation *string

	CompileRuntime *RuntimeData
}

type FullSubmission struct {
	Submission
	TestResults []EvalTestResult
	EvalDetails *EvalDetails
}

type SubmissionListUpdate struct {
	SubmCreated        *Submission
	StateUpdate        *SubmEvalStageUpdate
	TestgroupResUpdate *TestGroupScoringUpdate
	TestsResUpdate     *TestSetScoringUpdate
}

type SubmEvalStageUpdate struct {
	SubmUuid string
	EvalUuid string
	NewStage string
}

type TestGroupScoringUpdate struct {
	SubmUUID      string
	EvalUUID      string
	TestGroupID   int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

type TestSetScoringUpdate struct {
	SubmUuid string
	EvalUuid string
	Accepted int
	Wrong    int
	Untested int
}
