package subm

import "time"

type CreateSubmissionPayload struct {
	Submission        string
	Username          string
	ProgrammingLangID string
	TaskCodeID        string
	Token             string
}

type BriefSubmission struct {
	SubmUUID              string
	Username              string
	CreatedAt             string
	EvalUUID              string
	EvalStatus            string
	EvalScoringTestgroups []*TestGroupResult
	EvalScoringTests      *TestsResult
	EvalScoringSubtasks   []*SubtaskResult
	PLangID               string
	PLangDisplayName      string
	PLangMonacoID         string
	TaskName              string
	TaskID                string
}

type EvalTestResults struct {
	TestId   int
	Reached  bool
	Ignored  bool
	Finished bool

	InputTrimmed  *string
	AnswerTrimmed *string

	TimeLimitExceeded   *bool
	MemoryLimitExceeded *bool

	Subtasks  []int
	TestGroup *int

	SubmCpuTimeMillis *int
	SubmMemKibiBytes  *int
	SubmWallTime      *int
	SubmExitCode      *int
	SubmStdoutTrimmed *string
	SubmStderrTrimmed *string

	CheckerCpuTimeMillis *int
	CheckerMemKibiBytes  *int
	CheckerWallTime      *int
	CheckerExitCode      *int
	CheckerStdoutTrimmed *string
	CheckerStderrTrimmed *string
}

type EvalDetails struct {
	EvalUuid string

	CreatedAt time.Time
	ErrorMsg  *string
	EvalStage string

	CpuTimeLimitMillis   *int
	MemoryLimitKibiBytes *int

	ProgrammingLang   ProgrammingLang
	SystemInformation *string

	CompileCpuTimeMillis *int
	CompileMemKibiBytes  *int
	CompileWallTime      *int
	CompileExitCode      *int
	CompileStdoutTrimmed *string
	CompileStderrTrimmed *string
}

type FullSubmission struct {
	BriefSubmission
	SubmContent     string
	EvalTestResults []*EvalTestResults
	EvalDetails     *EvalDetails
}

type SubmissionListUpdate struct {
	SubmCreated        *BriefSubmission
	StateUpdate        *SubmissionStateUpdate
	TestgroupResUpdate *TestgroupScoreUpdate
}

type SubtaskResult struct {
	SubtaskID     int
	SubtaskScore  int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

type TestGroupResult struct {
	TestGroupID      int
	TestGroupScore   int
	StatementSubtask int
	AcceptedTests    int
	WrongTests       int
	UntestedTests    int
}

type TestgroupScoreUpdate struct {
	SubmUUID      string
	EvalUUID      string
	TestGroupID   int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

type TestsResult struct {
	Accepted int
	Wrong    int
	Untested int
}

type SubmissionStateUpdate struct {
	SubmUuid string
	EvalUuid string
	NewState string
}

type TestgroupResultUpdate struct {
	SubmUuid      string
	EvalUuid      string
	TestgroupId   int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}