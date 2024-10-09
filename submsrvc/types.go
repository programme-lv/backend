package submsrvc

import (
	"time"
)

type EvalTest struct {
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
	Submission
	TestResults []EvalTest
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
