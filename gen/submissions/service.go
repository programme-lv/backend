// Code generated by goa v3.18.2, DO NOT EDIT.
//
// submissions service
//
// Command:
// $ goa gen github.com/programme-lv/backend/design

package submissions

import (
	"context"

	goa "goa.design/goa/v3/pkg"
)

// Service for managing submissions
type Service interface {
	// Create a new submission
	CreateSubmission(context.Context, *CreateSubmissionPayload) (res *Submission, err error)
	// List all submissions
	ListSubmissions(context.Context) (res []*Submission, err error)
	// Get a submission by UUID
	GetSubmission(context.Context, *GetSubmissionPayload) (res *Submission, err error)
}

// APIName is the name of the API as defined in the design.
const APIName = "proglv"

// APIVersion is the version of the API as defined in the design.
const APIVersion = "0.0.1"

// ServiceName is the name of the service as defined in the design. This is the
// same value that is set in the endpoint request contexts under the ServiceKey
// key.
const ServiceName = "submissions"

// MethodNames lists the service method names as defined in the design. These
// are the same values that are set in the endpoint request contexts under the
// MethodKey key.
var MethodNames = [3]string{"createSubmission", "listSubmissions", "getSubmission"}

// CreateSubmissionPayload is the payload type of the submissions service
// createSubmission method.
type CreateSubmissionPayload struct {
	// The code submission
	Submission string
	// Username of the user who submitted
	Username string
	// ID of the programming language
	ProgrammingLangID string
	// ID of the task
	TaskCodeID string
}

// Represents the evaluation of a submission
type Evaluation struct {
	// UUID of the evaluation
	UUID string
	// Status of the evaluation
	Status string
	// Received score of the evaluation
	ReceivedScore int
	// Possible score of the evaluation
	PossibleScore int
}

// Represents an example for a task
type Example struct {
	// Example input
	Input string
	// Example output
	Output string
	// Markdown note for the example
	MdNote *string
}

// GetSubmissionPayload is the payload type of the submissions service
// getSubmission method.
type GetSubmissionPayload struct {
	// UUID of the submission
	UUID string
}

// Internal server error
type InternalError string

// Invalid submission details
type InvalidSubmissionDetails string

// Represents a markdown statement for a task
type MarkdownStatement struct {
	// Story section of the markdown statement
	Story string
	// Input section of the markdown statement
	Input string
	// Output section of the markdown statement
	Output string
	// Notes section of the markdown statement
	Notes *string
	// Scoring section of the markdown statement
	Scoring *string
}

// Submission not found
type NotFound string

// Represents a programming language
type ProgrammingLang struct {
	// ID of the programming language
	ID string
	// Full name of the programming language
	FullName string
	// Monaco editor ID for the programming language
	MonacoID string
}

// Represents subtask inputs for a task
type StInputs struct {
	// Subtask number
	Subtask int
	// Inputs for the subtask
	Inputs []string
}

// Submission is the result type of the submissions service createSubmission
// method.
type Submission struct {
	// UUID of the submission
	UUID string
	// The code submission
	Submission string
	// Username of the user who submitted
	Username string
	// Creation date of the submission
	CreatedAt string
	// Evaluation of the submission
	Evaluation *Evaluation
	// Programming language of the submission
	Language *ProgrammingLang
	// Task associated with the submission
	Task *Task
}

// Represents a competitive programming task
type Task struct {
	// ID of the published task
	PublishedTaskID string
	// Full name of the task
	TaskFullName string
	// Memory limit in megabytes
	MemoryLimitMegabytes int
	// CPU time limit in seconds
	CPUTimeLimitSeconds float64
	// Origin olympiad of the task
	OriginOlympiad string
	// URL of the illustration image
	IllustrationImgURL *string
	// Difficulty rating of the task
	DifficultyRating int
	// Default markdown statement of the task
	DefaultMdStatement *MarkdownStatement
	// Examples for the task
	Examples []*Example
	// URL of the default PDF statement
	DefaultPdfStatementURL *string
	// Origin notes for the task
	OriginNotes map[string]string
	// Visible input subtasks
	VisibleInputSubtasks []*StInputs
}

// Credentials are invalid
type Unauthorized string

// Error returns an error description.
func (e InternalError) Error() string {
	return "Internal server error"
}

// ErrorName returns "InternalError".
//
// Deprecated: Use GoaErrorName - https://github.com/goadesign/goa/issues/3105
func (e InternalError) ErrorName() string {
	return e.GoaErrorName()
}

// GoaErrorName returns "InternalError".
func (e InternalError) GoaErrorName() string {
	return "InternalError"
}

// Error returns an error description.
func (e InvalidSubmissionDetails) Error() string {
	return "Invalid submission details"
}

// ErrorName returns "InvalidSubmissionDetails".
//
// Deprecated: Use GoaErrorName - https://github.com/goadesign/goa/issues/3105
func (e InvalidSubmissionDetails) ErrorName() string {
	return e.GoaErrorName()
}

// GoaErrorName returns "InvalidSubmissionDetails".
func (e InvalidSubmissionDetails) GoaErrorName() string {
	return "InvalidSubmissionDetails"
}

// Error returns an error description.
func (e NotFound) Error() string {
	return "Submission not found"
}

// ErrorName returns "NotFound".
//
// Deprecated: Use GoaErrorName - https://github.com/goadesign/goa/issues/3105
func (e NotFound) ErrorName() string {
	return e.GoaErrorName()
}

// GoaErrorName returns "NotFound".
func (e NotFound) GoaErrorName() string {
	return "NotFound"
}

// Error returns an error description.
func (e Unauthorized) Error() string {
	return "Credentials are invalid"
}

// ErrorName returns "unauthorized".
//
// Deprecated: Use GoaErrorName - https://github.com/goadesign/goa/issues/3105
func (e Unauthorized) ErrorName() string {
	return e.GoaErrorName()
}

// GoaErrorName returns "unauthorized".
func (e Unauthorized) GoaErrorName() string {
	return "unauthorized"
}

// MakeInvalidSubmissionDetails builds a goa.ServiceError from an error.
func MakeInvalidSubmissionDetails(err error) *goa.ServiceError {
	return goa.NewServiceError(err, "InvalidSubmissionDetails", false, false, false)
}
