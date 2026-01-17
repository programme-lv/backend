package exec

import (
	"fmt"
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

type ExecStage string

const (
	StageWaiting       ExecStage = "waiting"
	StageCompiling     ExecStage = "compiling"
	StageTesting       ExecStage = "testing"
	StageFinished      ExecStage = "finished"
	StageCompileError  ExecStage = "compile_error"
	StageInternalError ExecStage = "internal_error"
)

type Execution struct {
	UUID      uuid.UUID     `json:"uuid"`
	Stage     ExecStage     `json:"stage"`
	TestRes   []TestRes     `json:"test_res"`
	PrLang    PrLang        `json:"pr_lang"`
	Params    TestingParams `json:"params"`
	ErrorMsg  *string       `json:"error_msg"`
	SysInfo   *string       `json:"sys_info"` // testing hardware info
	CreatedAt time.Time     `json:"created_at"`
	SubmComp  *RunData      `json:"subm_comp"` // submitted code compilation runtime data
	// ChecComp   *RunData // testlib checker compilation runtime data
}

// Does not compare UUID, SysInfo, CreatedAt
// Ensures that execution time deviations are within reasonable deviations
func (e *Execution) SimilarTo(other Execution) error {
	if e.Stage != other.Stage {
		return fmt.Errorf("stage mismatch: %s != %s", e.Stage, other.Stage)
	}
	if len(e.TestRes) != len(other.TestRes) {
		return fmt.Errorf("test results length mismatch: %d != %d", len(e.TestRes), len(other.TestRes))
	}
	for i := range e.TestRes {
		if err := e.TestRes[i].SimilarTo(other.TestRes[i]); err != nil {
			return fmt.Errorf("test result %d mismatch: %w", i, err)
		}
	}
	if e.PrLang != other.PrLang {
		return fmt.Errorf("programming language mismatch: %s != %s", e.PrLang.ShortId, other.PrLang.ShortId)
	}
	if (e.ErrorMsg == nil) != (other.ErrorMsg == nil) {
		return fmt.Errorf("error message nil mismatch: %v != %v", e.ErrorMsg, other.ErrorMsg)
	}
	if e.ErrorMsg != nil && other.ErrorMsg != nil && *e.ErrorMsg != *other.ErrorMsg {
		return fmt.Errorf("error message mismatch: %q != %q", *e.ErrorMsg, *other.ErrorMsg)
	}
	if (e.SubmComp == nil) != (other.SubmComp == nil) {
		return fmt.Errorf("subm_comp nil mismatch: %v != %v", e.SubmComp == nil, other.SubmComp == nil)
	}
	if e.SubmComp != nil && other.SubmComp != nil {
		if err := e.SubmComp.SimilarTo(*other.SubmComp); err != nil {
			return fmt.Errorf("subm_comp mismatch: %w", err)
		}
	}
	if e.Params != other.Params {
		return fmt.Errorf("testing params mismatch: %+v != %+v", e.Params, other.Params)
	}
	return nil
}

// Tester machine submitted solution runtime constraints
type TestingParams struct {
	CpuMs  int `json:"cpu_ms"`  // maximum user-mode CPU time in milliseconds
	MemKiB int `json:"mem_kib"` // maximum resident set size in kibibytes

	// optional testlib.h checker program. If not provided,
	// only output of the user's solution is returned from tester
	// and is not viable for grading. "run program" use case.
	Checker *string `json:"checker"`

	// optional testlib.h interactor program.
	Interactor *string `json:"interactor"`
}

func (p *TestingParams) IsValid() error {
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

	Subm    *RunData `json:"subm_rd"` // user submitted solution
	Checker *RunData `json:"tlib_rd"` // testlib checker
}

func (t *TestRes) SimilarTo(other TestRes) error {
	if t.ID != other.ID {
		return fmt.Errorf("test ID mismatch: %d != %d", t.ID, other.ID)
	}
	if (t.Input == nil) != (other.Input == nil) {
		return fmt.Errorf("input nil mismatch: %v != %v", t.Input == nil, other.Input == nil)
	}
	if t.Input != nil && other.Input != nil && *t.Input != *other.Input {
		return fmt.Errorf("input mismatch: %q != %q", *t.Input, *other.Input)
	}
	if (t.Answer == nil) != (other.Answer == nil) {
		return fmt.Errorf("answer nil mismatch: %v != %v", t.Answer == nil, other.Answer == nil)
	}
	if t.Answer != nil && other.Answer != nil && *t.Answer != *other.Answer {
		return fmt.Errorf("answer mismatch: %q != %q", *t.Answer, *other.Answer)
	}
	if t.Reached != other.Reached {
		return fmt.Errorf("reached mismatch: %v != %v", t.Reached, other.Reached)
	}
	if t.Ignored != other.Ignored {
		return fmt.Errorf("ignored mismatch: %v != %v", t.Ignored, other.Ignored)
	}
	if t.Finished != other.Finished {
		return fmt.Errorf("finished mismatch: %v != %v", t.Finished, other.Finished)
	}
	if (t.Subm == nil) != (other.Subm == nil) {
		return fmt.Errorf("subm nil mismatch: %v != %v", t.Subm == nil, other.Subm == nil)
	}
	if t.Subm != nil && other.Subm != nil {
		if err := t.Subm.SimilarTo(*other.Subm); err != nil {
			return fmt.Errorf("subm mismatch: %w", err)
		}
	}
	if (t.Checker == nil) != (other.Checker == nil) {
		return fmt.Errorf("checker nil mismatch: %v != %v", t.Checker == nil, other.Checker == nil)
	}
	if t.Checker != nil && other.Checker != nil {
		if err := t.Checker.SimilarTo(*other.Checker); err != nil {
			return fmt.Errorf("checker mismatch: %w", err)
		}
	}
	return nil
}

// Runtime Data
type RunData struct {
	StdIn       string  `json:"in"`            // standard input
	StdOut      string  `json:"out"`           // standard output
	StdErr      string  `json:"err"`           // standard error
	CpuMs       int64   `json:"cpu_ms"`        // cpu user-mode time in milliseconds
	WallMs      int64   `json:"wall_ms"`       // wall clock time in milliseconds
	MemKiB      int64   `json:"mem_kib"`       // memory usage (resident set size) in kibibytes
	ExitCode    int64   `json:"exit"`          // exit code
	CtxSwV      int64   `json:"ctx_sw_v"`      // voluntary context switches, e.g. waiting for I/O
	CtxSwF      int64   `json:"ctx_sw_f"`      // involuntary context switches, e.g. waiting for CPU
	Signal      *int64  `json:"signal"`        // exit signal if any
	IsOomKilled bool    `json:"is_oom_killed"` // whether the process was killed due to memory exhaustion
	IsolStatus  *string `json:"isol_status"`   // isolate sandbox execution environment status
	IsolMsg     *string `json:"isol_msg"`      // isolate sandbox execution environment message
}

func (r *RunData) SimilarTo(other RunData) error {
	if r.StdIn != other.StdIn {
		return fmt.Errorf("stdin mismatch: %q != %q", r.StdIn, other.StdIn)
	}
	if r.StdOut != other.StdOut {
		return fmt.Errorf("stdout mismatch: %q != %q", r.StdOut, other.StdOut)
	}
	if r.StdErr != other.StdErr {
		return fmt.Errorf("stderr mismatch: %q != %q", r.StdErr, other.StdErr)
	}

	// CPU time: 5% relative error or 20ms absolute, whichever is larger
	if !isWithinTolerance(r.CpuMs, other.CpuMs, 0.05, 20) {
		return fmt.Errorf("cpu_ms mismatch: %d != %d (tolerance: 5%% or 20ms)", r.CpuMs, other.CpuMs)
	}

	// Wall time: 10% relative error or 50ms absolute, whichever is larger (less predictable)
	if !isWithinTolerance(r.WallMs, other.WallMs, 0.10, 50) {
		return fmt.Errorf("wall_ms mismatch: %d != %d (tolerance: 10%% or 50ms)", r.WallMs, other.WallMs)
	}

	// Memory: 5% relative error or 5 MiB absolute, whichever is larger
	if !isWithinTolerance(r.MemKiB, other.MemKiB, 0.05, 5000) {
		return fmt.Errorf("mem_kib mismatch: %d != %d (tolerance: 5%% or 5 MiB)", r.MemKiB, other.MemKiB)
	}

	if r.ExitCode != other.ExitCode {
		return fmt.Errorf("exit_code mismatch: %d != %d", r.ExitCode, other.ExitCode)
	}

	// Context switches: 5% relative error or 10 switch absolute, whichever is larger
	if !isWithinTolerance(r.CtxSwV, other.CtxSwV, 0.05, 10) {
		return fmt.Errorf("ctx_sw_v mismatch: %d != %d (tolerance: 5%% or 1)", r.CtxSwV, other.CtxSwV)
	}
	if !isWithinTolerance(r.CtxSwF, other.CtxSwF, 0.05, 10) {
		return fmt.Errorf("ctx_sw_f mismatch: %d != %d (tolerance: 5%% or 1)", r.CtxSwF, other.CtxSwF)
	}

	if (r.Signal == nil) != (other.Signal == nil) {
		return fmt.Errorf("signal nil mismatch: %v != %v", r.Signal == nil, other.Signal == nil)
	}
	if r.Signal != nil && other.Signal != nil && *r.Signal != *other.Signal {
		return fmt.Errorf("signal mismatch: %d != %d", *r.Signal, *other.Signal)
	}

	if r.IsOomKilled != other.IsOomKilled {
		return fmt.Errorf("is_oom_killed mismatch: %v != %v", r.IsOomKilled, other.IsOomKilled)
	}

	if (r.IsolStatus == nil) != (other.IsolStatus == nil) {
		return fmt.Errorf("isol_status nil mismatch: %v != %v", r.IsolStatus == nil, other.IsolStatus == nil)
	}
	if r.IsolStatus != nil && other.IsolStatus != nil && *r.IsolStatus != *other.IsolStatus {
		return fmt.Errorf("isol_status mismatch: %q != %q", *r.IsolStatus, *other.IsolStatus)
	}

	if (r.IsolMsg == nil) != (other.IsolMsg == nil) {
		return fmt.Errorf("isol_msg nil mismatch: %v != %v", r.IsolMsg == nil, other.IsolMsg == nil)
	}
	if r.IsolMsg != nil && other.IsolMsg != nil && *r.IsolMsg != *other.IsolMsg {
		return fmt.Errorf("isol_msg mismatch: %q != %q", *r.IsolMsg, *other.IsolMsg)
	}

	return nil
}

// isWithinTolerance checks if two values are within tolerance of each other.
// Tolerance is the maximum of: relPercent% of expected, or absValue absolute difference.
func isWithinTolerance(expected, actual int64, relPercent float64, absValue int64) bool {
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}

	// Calculate relative tolerance
	relativeTol := int64(float64(expected) * relPercent)
	if relativeTol < 0 {
		relativeTol = -relativeTol
	}

	// Use the larger of relative or absolute tolerance
	tolerance := relativeTol
	if absValue > tolerance {
		tolerance = absValue
	}

	return diff <= tolerance
}

/*
time:0.002
time-wall:0.045
max-rss:2624
csw-voluntary:6
csw-forced:2
cg-mem:38248
exitcode:2
status:RE
message:Exited with error status 2
*/

// type Text struct {
// 	PvRect  string // preview rectangle cutout
// 	RectH   int    // rectangle max height
// 	RectW   int    // rectangle max width
// 	FullUrl string // full text access URL, likely stored in S3
// 	FullSz  int64  // full text size in bytes
// 	Sha256  string // SHA256 hash of full text
// }
