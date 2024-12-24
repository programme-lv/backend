package evalsrvc

import "time"

type Event interface {
	Type() string
}

const (
	StartedEvaluationType   = "started_evaluation"
	StartedCompilationType  = "started_compilation"
	FinishedCompilationType = "finished_compilation"
	StartedTestingType      = "started_testing"
	ReachedTestType         = "reached_test"
	FinishedTestType        = "finished_test"
	IgnoredTestType         = "ignored_test"
	FinishedTestingType     = "finished_testing"
	FinishedEvaluationType  = "finished_evaluation"
)

type StartedEvaluation struct {
	SysInfo   string    `json:"sys_info"`
	StartedAt time.Time `json:"started_at"`
}

func (s StartedEvaluation) Type() string {
	return StartedEvaluationType
}

type StartedCompiling struct{}

func (s StartedCompiling) Type() string {
	return StartedCompilationType
}

type FinishedCompiling struct {
	RuntimeData *RunData `json:"runtime_data"`
}

func (s FinishedCompiling) Type() string {
	return FinishedCompilationType
}

// Runtime Data
type RunData struct {
	StdIn    string `json:"in"`
	StdOut   string `json:"out"`
	StdErr   string `json:"err"`
	CpuMs    int64  `json:"cpu_ms"`
	WallMs   int64  `json:"wall_ms"`
	MemKiB   int64  `json:"mem_kib"`
	ExitCode int64  `json:"exit"`
	CtxSwV   *int64 `json:"ctx_sw_v"`
	CtxSwF   *int64 `json:"ctx_sw_f"`
	Signal   *int64 `json:"signal"`
}

type StartedTesting struct{}

func (s StartedTesting) Type() string {
	return StartedTestingType
}

type ReachedTest struct {
	TestId int     `json:"test_id"`
	In     *string `json:"in"`
	Ans    *string `json:"ans"`
}

func (s ReachedTest) Type() string {
	return ReachedTestType
}

type IgnoredTest struct {
	TestId int `json:"test_id"`
}

func (s IgnoredTest) Type() string {
	return IgnoredTestType
}

type FinishedTest struct {
	TestID  int      `json:"test_id"`
	Subm    *RunData `json:"submission"`
	Checker *RunData `json:"checker"`
}

func (s FinishedTest) Type() string {
	return FinishedTestType
}

type FinishedTesting struct{}

func (s FinishedTesting) Type() string {
	return FinishedTestingType
}

type FinishedEvaluation struct {
	CompileError  bool    `json:"compile_error"`
	InternalError bool    `json:"internal_error"`
	ErrorMsg      *string `json:"error_msg"`
}

func (s FinishedEvaluation) Type() string {
	return FinishedEvaluationType
}
