package execsrvc

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
	UUID      uuid.UUID    `json:"uuid"`
	Stage     string       `json:"stage"`
	TestRes   []TestRes    `json:"test_res"`
	PrLang    PrLang       `json:"pr_lang"`
	Params    TesterParams `json:"params"`
	ErrorMsg  *string      `json:"error_msg"`
	SysInfo   *string      `json:"sys_info"` // testing hardware info
	CreatedAt time.Time    `json:"created_at"`
	SubmComp  *RunData     `json:"subm_comp"` // submitted code compilation runtime data
	// ChecComp   *RunData // testlib checker compilation runtime data
}

// Tester machine submitted solution runtime constraints
type TesterParams struct {
	CpuMs  int `json:"cpu_ms"`  // maximum user-mode CPU time in milliseconds
	MemKiB int `json:"mem_kib"` // maximum resident set size in kibibytes

	// optional testlib.h checker program. If not provided,
	// only output of the user's solution is returned from tester
	// and is not viable for grading. "run program" use case.
	Checker *string `json:"checker"`

	// optional testlib.h interactor program.
	Interactor *string `json:"interactor"`
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
	ShortId   string  `json:"short_id"`   // short lang/compiler/interpreter id
	Display   string  `json:"display"`    // user-friendly programming lang name
	CodeFname string  `json:"code_fname"` // source code filename for mv in to box
	CompCmd   *string `json:"comp_cmd"`   // compile command
	CompFname *string `json:"comp_fname"` // exe fname after comp for mv out of box
	ExecCmd   string  `json:"exec_cmd"`   // execution command
}

type TestRes struct {
	ID       int     `json:"id"`
	Input    *string `json:"inp"` // trimmed test input file preview
	Answer   *string `json:"ans"` // trimmed test answer file preview
	Reached  bool    `json:"rch"`
	Ignored  bool    `json:"ign"` // when score group has another failed test
	Finished bool    `json:"fin"`

	ProgramReport *RunData `json:"subm_rd"` // user submitted solution
	CheckerReport *RunData `json:"tlib_rd"` // testlib checker
}

// Runtime Data
type RunData struct {
	StdIn      string  `json:"in"`          // standard input
	StdOut     string  `json:"out"`         // standard output
	StdErr     string  `json:"err"`         // standard error
	CpuMs      int64   `json:"cpu_ms"`      // cpu user-mode time in milliseconds
	WallMs     int64   `json:"wall_ms"`     // wall clock time in milliseconds
	MemKiB     int64   `json:"mem_kib"`     // memory usage (resident set size) in kibibytes
	ExitCode   int64   `json:"exit"`        // exit code
	CtxSwV     int64   `json:"ctx_sw_v"`    // voluntary context switches, e.g. waiting for I/O
	CtxSwF     int64   `json:"ctx_sw_f"`    // involuntary context switches, e.g. waiting for CPU
	Signal     *int64  `json:"signal"`      // exit signal if any
	IsolStatus *string `json:"isol_status"` // isolate sandbox execution environment status
}

// type Text struct {
// 	PvRect  string // preview rectangle cutout
// 	RectH   int    // rectangle max height
// 	RectW   int    // rectangle max width
// 	FullUrl string // full text access URL, likely stored in S3
// 	FullSz  int64  // full text size in bytes
// 	Sha256  string // SHA256 hash of full text
// }
