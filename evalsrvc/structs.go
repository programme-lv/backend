package evalsrvc

import (
	"time"

	"github.com/google/uuid"
)

// NewEvalParams contains parameters needed to create a new evaluation request
type NewEvalParams struct {
	Code   string // Source code to evaluate
	LangId string // Programming language identifier

	Tests []Test // Test cases to run against the code

	CpuMs  int // CPU time limit in milliseconds
	MemKiB int // Memory limit in KiB

	Checker    *string // Optional custom checker program
	Interactor *string // Optional interactive testing program
}

// Test represents a single test case with input and expected output
type Test struct {
	ID int // Test case identifier

	InSha256  *string // SHA256 hash of input file for verification
	InUrl     *string // URL to download input file
	InContent *string // Raw input content if not using URL

	AnsSha256  *string // SHA256 hash of answer file for verification
	AnsUrl     *string // URL to download answer file
	AnsContent *string // Raw answer content if not using URL
}

type Evaluation struct {
	UUID       uuid.UUID
	Stage      string
	Tests      []TestRes
	PrLang     PrLang
	ErrorMsg   *string
	Checker    *string
	Interactor *string
	SysInfo    *string // testing hardware info
	CpuMsLim   int
	MemKiBLim  int
	CreatedAt  time.Time
	SubmComp   *RuntimeData // user submitted solution compile runtime data
	ChecComp   *RuntimeData // testlib checker compile runtime data
}

type PrLang struct {
	ShortId   string
	FullName  string
	CodeFname string
	CompCmd   *string
	CompFname *string
	ExecCmd   string
}

type TestRes struct {
	ID       int
	InpUrl   *string
	AnsUrl   *string
	InpShort *string // input preview
	AnsShort *string // answer preview
	Reached  bool
	Ignored  bool
	Finished bool
	Checker  *RuntimeData // testlib checker runtime data
	Program  *RuntimeData // user submitted solution runtime data
}

type RuntimeData struct {
	StdoutShort *string
	StderrShort *string
	StdoutUrl   *string // full stdout, possibly in S3
	StderrUrl   *string // full stderr, possibly in S3
	ExitCode    int64
	CPUTime     int64
	WallTime    int64
	Memory      int64
	CtxSwForced *int64
	ExitSignal  *int64
	IsolStatus  *string // isolate execution environment status
}
