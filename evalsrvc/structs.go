package evalsrvc

import (
	"time"

	"github.com/google/uuid"
)

const (
	EvalStageWaiting   = "waiting"
	EvalStageCompiling = "compiling"
	EvalStageTesting   = "testing"
	EvalStageDone      = "done"
	EvalStageFailed    = "failed"
)

type Evaluation struct {
	UUID      uuid.UUID
	Stage     string
	TestRes   []TestRes
	PrLang    PrLang
	Params    TesterParams
	ErrorMsg  *string
	SysInfo   *string // testing hardware info
	CreatedAt time.Time
	SubmComp  *RuntimeData // submitted code compilation runtime data
	// ChecComp   *RuntimeData // testlib checker compilation runtime data
}

// Tester machine submitted solution runtime constraints
type TesterParams struct {
	CpuMs  int // maximum user-mode CPU time in milliseconds
	MemKiB int // maximum resident set size in kibibytes

	// optional testlib.h checker program. If not provided,
	// only output of the user's solution is returned from tester
	// and is not viable for grading. "run program" use case.
	Checker *string

	// optional testlib.h interactor program.
	Interactor *string
}

func (p *TesterParams) IsValid() error {
	if p.CpuMs <= 0 {
		return ErrInvalidTesterParams()
	}
	if p.MemKiB <= 0 {
		return ErrInvalidTesterParams()
	}
	if p.CpuMs > 10*1000 { // 10 seconds
		return ErrCpuConstraintTooLose()
	}
	if p.MemKiB > 1024*1024 { // 1 GiB
		return ErrMemConstraintTooLose()
	}
	if p.Checker != nil && len(*p.Checker) > 1024*1024 { // 1 MiB
		return ErrCheckerTooLarge()
	}
	if p.Interactor != nil && len(*p.Interactor) > 1024*1024 { // 1 MiB
		return ErrInteractorTooLarge()
	}
	return nil
}

type PrLang struct {
	ShortId   string  // short lang/compiler/interpreter id
	Display   string  // user-friendly programming lang name
	CodeFname string  // source code filename for mv in to box
	CompCmd   *string // compile command
	CompFname *string // exe fname after comp for mv out of box
	ExecCmd   string  // execution command
}

type TestRes struct {
	ID       int
	Input    *Text // test input
	Answer   *Text // test answer
	Reached  bool
	Ignored  bool // when score group has another failed test
	Finished bool

	CheckerReport *RuntimeData // testlib checker
	ProgramReport *RuntimeData // user submitted solution
}

type RuntimeData struct {
	StdIn  *Text
	StdOut *Text
	StdErr *Text

	ExitCode    int64
	CpuTimeMs   int64   // cpu user-mode time in milliseconds
	WallTimeMs  int64   // wall clock time in milliseconds
	MemoryKiB   int64   // memory usage in kibibytes
	CtxSwForced *int64  // context switches forced by kernel
	ExitSignal  *int64  // exit signal if any
	IsolStatus  *string // isolate execution environment status
}

type Text struct {
	PvRect  string // preview rectangle cutout
	RectH   int    // rectangle max height
	RectW   int    // rectangle max width
	FullUrl string // full text access URL, likely stored in S3
	FullSz  int64  // full text size in bytes
	Sha256  string // SHA256 hash of full text
}
