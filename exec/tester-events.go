package exec

import (
	"encoding/json"
	"reflect"
	"strings"
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

// eventToMap converts an event struct to a map[string]interface{} by extracting
// all JSON-tagged fields and adding the type field at the top level.
func eventToMap(event Event, data interface{}) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["type"] = event.Type()

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse the tag to handle cases like `json:"field_name,omitempty"`
		tagName := strings.Split(jsonTag, ",")[0]
		if tagName == "" {
			continue
		}

		fieldValue := v.Field(i)
		m[tagName] = fieldValue.Interface()
	}

	return m, nil
}

type ReceivedSubmission struct {
	SysInfo   string    `json:"sys_info"`
	StartedAt time.Time `json:"started_at"`
}

var _ Event = ReceivedSubmission{}

func (s ReceivedSubmission) Type() string {
	return ReceivedSubmissionType
}

func (s ReceivedSubmission) MarshalJSON() ([]byte, error) {
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

type StartedCompiling struct{}

var _ Event = StartedCompiling{}

func (s StartedCompiling) Type() string {
	return StartedCompilationType
}

func (s StartedCompiling) MarshalJSON() ([]byte, error) {
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

type FinishedCompiling struct {
	RuntimeData *RunData `json:"runtime_data"`
}

var _ Event = FinishedCompiling{}

func (s FinishedCompiling) Type() string {
	return FinishedCompilationType
}

func (s FinishedCompiling) MarshalJSON() ([]byte, error) {
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
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
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

type IgnoredTest struct {
	TestId int `json:"test_id"`
}

var _ Event = IgnoredTest{}

func (s IgnoredTest) Type() string {
	return IgnoredTestType
}

func (s IgnoredTest) MarshalJSON() ([]byte, error) {
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
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
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

type FinishedTesting struct{}

var _ Event = FinishedTesting{}

func (s FinishedTesting) Type() string {
	return FinishedTestingType
}

func (s FinishedTesting) MarshalJSON() ([]byte, error) {
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

type CompilationError struct {
	ErrorMsg *string `json:"error_msg"`
}

var _ Event = CompilationError{}

func (s CompilationError) Type() string {
	return CompilationErrorType
}

func (s CompilationError) MarshalJSON() ([]byte, error) {
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

type InternalServerError struct {
	ErrorMsg *string `json:"error_msg"`
}

var _ Event = InternalServerError{}

func (s InternalServerError) Type() string {
	return InternalServerErrorType
}

func (s InternalServerError) MarshalJSON() ([]byte, error) {
	m, err := eventToMap(s, s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}
