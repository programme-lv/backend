package evalsrvc

import "time"

type Event interface {
	Type() string
}

const (
	ReceivedSubmissionType  = "received_submission"
	StartedCompilationType  = "started_compilation"
	FinishedCompilationType = "finished_compilation"
	CompilationErrorType    = "compilation_error"
	StartedTestingType      = "started_testing"
	ReachedTestType         = "reached_test"
	IgnoredTestType         = "ignored_test"
	FinishedTestType        = "finished_test"
	FinishedTestingType     = "finished_testing"
	InternalServerErrorType = "internal_server_error"
)

type ReceivedSubmission struct {
	SysInfo   string    `json:"sys_info"`
	StartedAt time.Time `json:"started_at"`
}

func (s ReceivedSubmission) Type() string {
	return ReceivedSubmissionType
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

type CompilationError struct {
	ErrorMsg *string `json:"error_msg"`
}

func (s CompilationError) Type() string {
	return CompilationErrorType
}

type InternalServerError struct {
	ErrorMsg *string `json:"error_msg"`
}

func (s InternalServerError) Type() string {
	return InternalServerErrorType
}
