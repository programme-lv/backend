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
	SysInfo   string
	StartedAt time.Time
}

func (s StartedEvaluation) Type() string {
	return MsgTypeStartedEvaluation
}

type StartedCompiling struct{}

func (s StartedCompiling) Type() string {
	return MsgTypeStartedCompilation
}

type FinishedCompiling struct {
	RuntimeData *RunData
}

func (s FinishedCompiling) Type() string {
	return MsgTypeFinishedCompilation
}

// Runtime Data
type RunData struct {
	StdIn, StdOut, StdErr string
	CpuMs, WallMs, MemKiB int64
	ExitCode              int64
	CtxSwV, CtxSwF        int64
	Signal                *int64
}

type StartedTesting struct{}

func (s StartedTesting) Type() string {
	return MsgTypeStartedTesting
}

type ReachedTest struct {
	TestId  int64
	In, Ans *string
}

func (s ReachedTest) Type() string {
	return MsgTypeReachedTest
}

type IgnoredTest struct {
	TestId int64
}

func (s IgnoredTest) Type() string {
	return MsgTypeIgnoredTest
}

type FinishedTest struct {
	TestID  int64
	Subm    *RunData
	Checker *RunData
}

func (s FinishedTest) Type() string {
	return MsgTypeFinishedTest
}

type FinishedTesting struct{}

func (s FinishedTesting) Type() string {
	return MsgTypeFinishedTesting
}

type FinishedEvaluation struct {
	CompileError  bool
	InternalError bool
	ErrorMsg      *string
}

func (s FinishedEvaluation) Type() string {
	return MsgTypeFinishedEvaluation
}
