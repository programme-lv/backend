package evalsrvc

import (
	"time"

	"github.com/google/uuid"
)

// user submitted solution
type CodeWithLang struct {
	SrcCode string // user submitted solution source code
	LangId  string // short compiler, interpreter id
}

// input and expected output
type TestFile struct {
	InSha256   *string // SHA256 hash of input for caching
	InDownlUrl *string // URL to download input
	InContent  *string // input content as alternative to URL

	AnsSha256   *string // SHA256 hash of answer for caching
	AnsDownlUrl *string // URL to download answer
	AnsContent  *string // answer content as alternative to URL
}

func (t *TestFile) IsValid() error {
	if t.InContent == nil && (t.InSha256 == nil || t.InDownlUrl == nil) {
		return ErrInvalidTestFile()
	}
	if t.AnsContent == nil && (t.AnsSha256 == nil || t.AnsDownlUrl == nil) {
		return ErrInvalidTestFile()
	}
	return nil
}

const (
	StageWaiting       = "waiting"
	StageCompiling     = "compiling"
	StageTesting       = "testing"
	StageFinished      = "finished"
	StageCompileError  = "compile_error"
	StageInternalError = "internal_error"
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
	SubmComp  *RunData // submitted code compilation runtime data
	// ChecComp   *RunData // testlib checker compilation runtime data
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
	Input    *string // test input
	Answer   *string // test answer
	Reached  bool
	Ignored  bool // when score group has another failed test
	Finished bool

	CheckerReport *RunData // testlib checker
	ProgramReport *RunData // user submitted solution
}

// Runtime Data
type RunData struct {
	StdIn    string `json:"in"`
	StdOut   string `json:"out"`
	StdErr   string `json:"err"`
	CpuMs    int64  `json:"cpu_ms"`   // cpu user-mode time in milliseconds
	WallMs   int64  `json:"wall_ms"`  // wall clock time in milliseconds
	MemKiB   int64  `json:"mem_kib"`  // memory usage in kibibytes
	ExitCode int64  `json:"exit"`     // exit code
	CtxSwV   int64  `json:"ctx_sw_v"` // context switches forced by kernel
	CtxSwF   int64  `json:"ctx_sw_f"` // context switches forced by kernel
	Signal   *int64 `json:"signal"`   // exit signal if any
	// IsolStatus *string `json:"isol_status"` // isolate execution environment status
}

// type Text struct {
// 	PvRect  string // preview rectangle cutout
// 	RectH   int    // rectangle max height
// 	RectW   int    // rectangle max width
// 	FullUrl string // full text access URL, likely stored in S3
// 	FullSz  int64  // full text size in bytes
// 	Sha256  string // SHA256 hash of full text
// }
