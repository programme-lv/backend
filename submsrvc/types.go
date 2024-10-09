package submsrvc

import "time"

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
	SubmExitSignal    *int

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
	StateUpdate        *SubmEvalStageUpdate
	TestgroupResUpdate *TestgroupScoreUpdate
	TestsResUpdate     *TestScoreUpdate
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

type SubmEvalStageUpdate struct {
	SubmUuid string
	EvalUuid string
	NewStage string
}

type TestGroupScoreUpdate struct {
	SubmUuid      string
	EvalUuid      string
	TestgroupId   int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

type TestScoreUpdate struct {
	SubmUuid string
	EvalUuid string
	Accepted int
	Wrong    int
	Untested int
}
