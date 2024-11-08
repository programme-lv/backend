package submsrvc

import (
	"time"
)

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
	StateUpdate        *EvalStageUpd
	TestgroupResUpdate *TGroupScoreUpd
	TestsResUpdate     *TSetScoreUpd
}

type EvalStageUpd struct {
	SubmUuid string
	EvalUuid string
	NewStage string
}

type TGroupScoreUpd struct {
	SubmUUID      string
	EvalUUID      string
	TestGroupID   int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

type TSetScoreUpd struct {
	SubmUuid string
	EvalUuid string
	Accepted int
	Wrong    int
	Untested int
}
