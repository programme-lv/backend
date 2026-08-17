package exec

import (
	"encoding/json"
	"time"
)

type Event interface {
	Type() string
	MarshalJSON() ([]byte, error)
}

const (
	ReceivedSubmissionType  = "received_submission"
	StartedCompilationType  = "started_compilation"
	FinishedCompilationType = "finished_compilation"
	CompilationErrorType    = "compilation_error"
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

var _ Event = ReceivedSubmission{}

func (s ReceivedSubmission) Type() string {
	return ReceivedSubmissionType
}

func (s ReceivedSubmission) MarshalJSON() ([]byte, error) {
	type payload ReceivedSubmission
	return json.Marshal(struct {
		Type string `json:"type"`
		payload
	}{Type: s.Type(), payload: payload(s)})
}

type StartedCompiling struct{}

var _ Event = StartedCompiling{}

func (s StartedCompiling) Type() string {
	return StartedCompilationType
}

func (s StartedCompiling) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
	}{Type: s.Type()})
}

type FinishedCompiling struct {
	RuntimeData *RunData `json:"runtime_data"`
}

var _ Event = FinishedCompiling{}

func (s FinishedCompiling) Type() string {
	return FinishedCompilationType
}

func (s FinishedCompiling) MarshalJSON() ([]byte, error) {
	type payload FinishedCompiling
	return json.Marshal(struct {
		Type string `json:"type"`
		payload
	}{Type: s.Type(), payload: payload(s)})
}

type ReachedTest struct {
	TestId int     `json:"test_id"`
	In     *string `json:"in"`
	Ans    *string `json:"ans"`
}

var _ Event = ReachedTest{}

func (s ReachedTest) Type() string {
	return ReachedTestType
}

func (s ReachedTest) MarshalJSON() ([]byte, error) {
	type payload ReachedTest
	return json.Marshal(struct {
		Type string `json:"type"`
		payload
	}{Type: s.Type(), payload: payload(s)})
}

type IgnoredTest struct {
	TestId int `json:"test_id"`
}

var _ Event = IgnoredTest{}

func (s IgnoredTest) Type() string {
	return IgnoredTestType
}

func (s IgnoredTest) MarshalJSON() ([]byte, error) {
	type payload IgnoredTest
	return json.Marshal(struct {
		Type string `json:"type"`
		payload
	}{Type: s.Type(), payload: payload(s)})
}

type FinishedTest struct {
	TestID  int      `json:"test_id"`
	Subm    *RunData `json:"submission"`
	Checker *RunData `json:"checker"`
}

var _ Event = FinishedTest{}

func (s FinishedTest) Type() string {
	return FinishedTestType
}

func (s FinishedTest) MarshalJSON() ([]byte, error) {
	type payload FinishedTest
	return json.Marshal(struct {
		Type string `json:"type"`
		payload
	}{Type: s.Type(), payload: payload(s)})
}

type FinishedTesting struct{}

var _ Event = FinishedTesting{}

func (s FinishedTesting) Type() string {
	return FinishedTestingType
}

func (s FinishedTesting) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
	}{Type: s.Type()})
}

type CompilationError struct {
	ErrorMsg *string `json:"error_msg"`
}

var _ Event = CompilationError{}

func (s CompilationError) Type() string {
	return CompilationErrorType
}

func (s CompilationError) MarshalJSON() ([]byte, error) {
	type payload CompilationError
	return json.Marshal(struct {
		Type string `json:"type"`
		payload
	}{Type: s.Type(), payload: payload(s)})
}

type InternalServerError struct {
	ErrorMsg *string `json:"error_msg"`
}

var _ Event = InternalServerError{}

func (s InternalServerError) Type() string {
	return InternalServerErrorType
}

func (s InternalServerError) MarshalJSON() ([]byte, error) {
	type payload InternalServerError
	return json.Marshal(struct {
		Type string `json:"type"`
		payload
	}{Type: s.Type(), payload: payload(s)})
}
