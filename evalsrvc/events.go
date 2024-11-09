package evalsrvc

import "time"

const (
	MsgTypeStartedEvaluation   = "started_evaluation"
	MsgTypeStartedCompilation  = "started_compilation"
	MsgTypeFinishedCompilation = "finished_compilation"
	MsgTypeStartedTesting      = "started_testing"
	MsgTypeReachedTest         = "reached_test"
	MsgTypeFinishedTest        = "finished_test"
	MsgTypeIgnoredTest         = "ignored_test"
	MsgTypeFinishedTesting     = "finished_testing"
	MsgTypeFinishedEvaluation  = "finished_evaluation"
)

type StartedEvaluation struct {
	SysInfo   string    `json:"sys_info"`
	StartedAt time.Time `json:"started_at"`
}

func (s StartedEvaluation) Type() string {
	return MsgTypeStartedEvaluation
}

type StartedCompiling struct{}

func (s StartedCompiling) Type() string {
	return MsgTypeStartedCompilation
}

type FinishedCompiling struct {
	RuntimeData *RunData `json:"runtime_data"`
}

func (s FinishedCompiling) Type() string {
	return MsgTypeFinishedCompilation
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
	CtxSwV   int64  `json:"ctx_sw_v"`
	CtxSwF   int64  `json:"ctx_sw_f"`
	Signal   *int64 `json:"signal"`
}

type StartedTesting struct{}

func (s StartedTesting) Type() string {
	return MsgTypeStartedTesting
}

type ReachedTest struct {
	TestId int64   `json:"test_id"`
	In     *string `json:"in"`
	Ans    *string `json:"ans"`
}

func (s ReachedTest) Type() string {
	return MsgTypeReachedTest
}

type IgnoredTest struct {
	TestId int64 `json:"test_id"`
}

func (s IgnoredTest) Type() string {
	return MsgTypeIgnoredTest
}

type FinishedTest struct {
	TestID  int64    `json:"test_id"`
	Subm    *RunData `json:"submission"`
	Checker *RunData `json:"checker"`
}

func (s FinishedTest) Type() string {
	return MsgTypeFinishedTest
}

type FinishedTesting struct{}

func (s FinishedTesting) Type() string {
	return MsgTypeFinishedTesting
}

type FinishedEvaluation struct {
	CompileError  bool    `json:"compile_error"`
	InternalError bool    `json:"internal_error"`
	ErrorMsg      *string `json:"error_msg"`
}

func (s FinishedEvaluation) Type() string {
	return MsgTypeFinishedEvaluation
}
