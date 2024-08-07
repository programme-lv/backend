// Code generated by goa v3.18.2, DO NOT EDIT.
//
// tasks HTTP client types
//
// Command:
// $ goa gen github.com/programme-lv/backend/design

package client

import (
	tasks "github.com/programme-lv/backend/gen/tasks"
	goa "goa.design/goa/v3/pkg"
)

// ListTasksResponseBody is the type of the "tasks" service "listTasks"
// endpoint HTTP response body.
type ListTasksResponseBody []*TaskResponse

// GetTaskResponseBody is the type of the "tasks" service "getTask" endpoint
// HTTP response body.
type GetTaskResponseBody struct {
	// ID of the published task
	PublishedTaskID *string `form:"published_task_id,omitempty" json:"published_task_id,omitempty" xml:"published_task_id,omitempty"`
	// Full name of the task
	TaskFullName *string `form:"task_full_name,omitempty" json:"task_full_name,omitempty" xml:"task_full_name,omitempty"`
	// Memory limit in megabytes
	MemoryLimitMegabytes *int `form:"memory_limit_megabytes,omitempty" json:"memory_limit_megabytes,omitempty" xml:"memory_limit_megabytes,omitempty"`
	// CPU time limit in seconds
	CPUTimeLimitSeconds *float64 `form:"cpu_time_limit_seconds,omitempty" json:"cpu_time_limit_seconds,omitempty" xml:"cpu_time_limit_seconds,omitempty"`
	// Origin olympiad of the task
	OriginOlympiad *string `form:"origin_olympiad,omitempty" json:"origin_olympiad,omitempty" xml:"origin_olympiad,omitempty"`
	// URL of the illustration image
	IllustrationImgURL *string `form:"illustration_img_url,omitempty" json:"illustration_img_url,omitempty" xml:"illustration_img_url,omitempty"`
	// Difficulty rating of the task
	DifficultyRating *int `form:"difficulty_rating,omitempty" json:"difficulty_rating,omitempty" xml:"difficulty_rating,omitempty"`
	// Default markdown statement of the task
	DefaultMdStatement *MarkdownStatementResponseBody `form:"default_md_statement,omitempty" json:"default_md_statement,omitempty" xml:"default_md_statement,omitempty"`
	// Examples for the task
	Examples []*ExampleResponseBody `form:"examples,omitempty" json:"examples,omitempty" xml:"examples,omitempty"`
	// URL of the default PDF statement
	DefaultPdfStatementURL *string `form:"default_pdf_statement_url,omitempty" json:"default_pdf_statement_url,omitempty" xml:"default_pdf_statement_url,omitempty"`
	// Origin notes for the task
	OriginNotes map[string]string `form:"origin_notes,omitempty" json:"origin_notes,omitempty" xml:"origin_notes,omitempty"`
	// Visible input subtasks
	VisibleInputSubtasks []*StInputsResponseBody `form:"visible_input_subtasks,omitempty" json:"visible_input_subtasks,omitempty" xml:"visible_input_subtasks,omitempty"`
}

// TaskResponse is used to define fields on response body types.
type TaskResponse struct {
	// ID of the published task
	PublishedTaskID *string `form:"published_task_id,omitempty" json:"published_task_id,omitempty" xml:"published_task_id,omitempty"`
	// Full name of the task
	TaskFullName *string `form:"task_full_name,omitempty" json:"task_full_name,omitempty" xml:"task_full_name,omitempty"`
	// Memory limit in megabytes
	MemoryLimitMegabytes *int `form:"memory_limit_megabytes,omitempty" json:"memory_limit_megabytes,omitempty" xml:"memory_limit_megabytes,omitempty"`
	// CPU time limit in seconds
	CPUTimeLimitSeconds *float64 `form:"cpu_time_limit_seconds,omitempty" json:"cpu_time_limit_seconds,omitempty" xml:"cpu_time_limit_seconds,omitempty"`
	// Origin olympiad of the task
	OriginOlympiad *string `form:"origin_olympiad,omitempty" json:"origin_olympiad,omitempty" xml:"origin_olympiad,omitempty"`
	// URL of the illustration image
	IllustrationImgURL *string `form:"illustration_img_url,omitempty" json:"illustration_img_url,omitempty" xml:"illustration_img_url,omitempty"`
	// Difficulty rating of the task
	DifficultyRating *int `form:"difficulty_rating,omitempty" json:"difficulty_rating,omitempty" xml:"difficulty_rating,omitempty"`
	// Default markdown statement of the task
	DefaultMdStatement *MarkdownStatementResponse `form:"default_md_statement,omitempty" json:"default_md_statement,omitempty" xml:"default_md_statement,omitempty"`
	// Examples for the task
	Examples []*ExampleResponse `form:"examples,omitempty" json:"examples,omitempty" xml:"examples,omitempty"`
	// URL of the default PDF statement
	DefaultPdfStatementURL *string `form:"default_pdf_statement_url,omitempty" json:"default_pdf_statement_url,omitempty" xml:"default_pdf_statement_url,omitempty"`
	// Origin notes for the task
	OriginNotes map[string]string `form:"origin_notes,omitempty" json:"origin_notes,omitempty" xml:"origin_notes,omitempty"`
	// Visible input subtasks
	VisibleInputSubtasks []*StInputsResponse `form:"visible_input_subtasks,omitempty" json:"visible_input_subtasks,omitempty" xml:"visible_input_subtasks,omitempty"`
}

// MarkdownStatementResponse is used to define fields on response body types.
type MarkdownStatementResponse struct {
	// Story section of the markdown statement
	Story *string `form:"story,omitempty" json:"story,omitempty" xml:"story,omitempty"`
	// Input section of the markdown statement
	Input *string `form:"input,omitempty" json:"input,omitempty" xml:"input,omitempty"`
	// Output section of the markdown statement
	Output *string `form:"output,omitempty" json:"output,omitempty" xml:"output,omitempty"`
	// Notes section of the markdown statement
	Notes *string `form:"notes,omitempty" json:"notes,omitempty" xml:"notes,omitempty"`
	// Scoring section of the markdown statement
	Scoring *string `form:"scoring,omitempty" json:"scoring,omitempty" xml:"scoring,omitempty"`
}

// ExampleResponse is used to define fields on response body types.
type ExampleResponse struct {
	// Example input
	Input *string `form:"input,omitempty" json:"input,omitempty" xml:"input,omitempty"`
	// Example output
	Output *string `form:"output,omitempty" json:"output,omitempty" xml:"output,omitempty"`
	// Markdown note for the example
	MdNote *string `form:"md_note,omitempty" json:"md_note,omitempty" xml:"md_note,omitempty"`
}

// StInputsResponse is used to define fields on response body types.
type StInputsResponse struct {
	// Subtask number
	Subtask *int `form:"subtask,omitempty" json:"subtask,omitempty" xml:"subtask,omitempty"`
	// Inputs for the subtask
	Inputs []string `form:"inputs,omitempty" json:"inputs,omitempty" xml:"inputs,omitempty"`
}

// MarkdownStatementResponseBody is used to define fields on response body
// types.
type MarkdownStatementResponseBody struct {
	// Story section of the markdown statement
	Story *string `form:"story,omitempty" json:"story,omitempty" xml:"story,omitempty"`
	// Input section of the markdown statement
	Input *string `form:"input,omitempty" json:"input,omitempty" xml:"input,omitempty"`
	// Output section of the markdown statement
	Output *string `form:"output,omitempty" json:"output,omitempty" xml:"output,omitempty"`
	// Notes section of the markdown statement
	Notes *string `form:"notes,omitempty" json:"notes,omitempty" xml:"notes,omitempty"`
	// Scoring section of the markdown statement
	Scoring *string `form:"scoring,omitempty" json:"scoring,omitempty" xml:"scoring,omitempty"`
}

// ExampleResponseBody is used to define fields on response body types.
type ExampleResponseBody struct {
	// Example input
	Input *string `form:"input,omitempty" json:"input,omitempty" xml:"input,omitempty"`
	// Example output
	Output *string `form:"output,omitempty" json:"output,omitempty" xml:"output,omitempty"`
	// Markdown note for the example
	MdNote *string `form:"md_note,omitempty" json:"md_note,omitempty" xml:"md_note,omitempty"`
}

// StInputsResponseBody is used to define fields on response body types.
type StInputsResponseBody struct {
	// Subtask number
	Subtask *int `form:"subtask,omitempty" json:"subtask,omitempty" xml:"subtask,omitempty"`
	// Inputs for the subtask
	Inputs []string `form:"inputs,omitempty" json:"inputs,omitempty" xml:"inputs,omitempty"`
}

// NewListTasksTaskOK builds a "tasks" service "listTasks" endpoint result from
// a HTTP "OK" response.
func NewListTasksTaskOK(body []*TaskResponse) []*tasks.Task {
	v := make([]*tasks.Task, len(body))
	for i, val := range body {
		v[i] = unmarshalTaskResponseToTasksTask(val)
	}

	return v
}

// NewGetTaskTaskOK builds a "tasks" service "getTask" endpoint result from a
// HTTP "OK" response.
func NewGetTaskTaskOK(body *GetTaskResponseBody) *tasks.Task {
	v := &tasks.Task{
		PublishedTaskID:        *body.PublishedTaskID,
		TaskFullName:           *body.TaskFullName,
		MemoryLimitMegabytes:   *body.MemoryLimitMegabytes,
		CPUTimeLimitSeconds:    *body.CPUTimeLimitSeconds,
		OriginOlympiad:         *body.OriginOlympiad,
		IllustrationImgURL:     body.IllustrationImgURL,
		DifficultyRating:       *body.DifficultyRating,
		DefaultPdfStatementURL: body.DefaultPdfStatementURL,
	}
	if body.DefaultMdStatement != nil {
		v.DefaultMdStatement = unmarshalMarkdownStatementResponseBodyToTasksMarkdownStatement(body.DefaultMdStatement)
	}
	if body.Examples != nil {
		v.Examples = make([]*tasks.Example, len(body.Examples))
		for i, val := range body.Examples {
			v.Examples[i] = unmarshalExampleResponseBodyToTasksExample(val)
		}
	}
	if body.OriginNotes != nil {
		v.OriginNotes = make(map[string]string, len(body.OriginNotes))
		for key, val := range body.OriginNotes {
			tk := key
			tv := val
			v.OriginNotes[tk] = tv
		}
	}
	if body.VisibleInputSubtasks != nil {
		v.VisibleInputSubtasks = make([]*tasks.StInputs, len(body.VisibleInputSubtasks))
		for i, val := range body.VisibleInputSubtasks {
			v.VisibleInputSubtasks[i] = unmarshalStInputsResponseBodyToTasksStInputs(val)
		}
	}

	return v
}

// ValidateGetTaskResponseBody runs the validations defined on
// GetTaskResponseBody
func ValidateGetTaskResponseBody(body *GetTaskResponseBody) (err error) {
	if body.PublishedTaskID == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("published_task_id", "body"))
	}
	if body.TaskFullName == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("task_full_name", "body"))
	}
	if body.MemoryLimitMegabytes == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("memory_limit_megabytes", "body"))
	}
	if body.CPUTimeLimitSeconds == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("cpu_time_limit_seconds", "body"))
	}
	if body.OriginOlympiad == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("origin_olympiad", "body"))
	}
	if body.DifficultyRating == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("difficulty_rating", "body"))
	}
	if body.DifficultyRating != nil {
		if !(*body.DifficultyRating == 1 || *body.DifficultyRating == 2 || *body.DifficultyRating == 3 || *body.DifficultyRating == 4 || *body.DifficultyRating == 5) {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.difficulty_rating", *body.DifficultyRating, []any{1, 2, 3, 4, 5}))
		}
	}
	if body.DefaultMdStatement != nil {
		if err2 := ValidateMarkdownStatementResponseBody(body.DefaultMdStatement); err2 != nil {
			err = goa.MergeErrors(err, err2)
		}
	}
	for _, e := range body.Examples {
		if e != nil {
			if err2 := ValidateExampleResponseBody(e); err2 != nil {
				err = goa.MergeErrors(err, err2)
			}
		}
	}
	for _, e := range body.VisibleInputSubtasks {
		if e != nil {
			if err2 := ValidateStInputsResponseBody(e); err2 != nil {
				err = goa.MergeErrors(err, err2)
			}
		}
	}
	return
}

// ValidateTaskResponse runs the validations defined on TaskResponse
func ValidateTaskResponse(body *TaskResponse) (err error) {
	if body.PublishedTaskID == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("published_task_id", "body"))
	}
	if body.TaskFullName == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("task_full_name", "body"))
	}
	if body.MemoryLimitMegabytes == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("memory_limit_megabytes", "body"))
	}
	if body.CPUTimeLimitSeconds == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("cpu_time_limit_seconds", "body"))
	}
	if body.OriginOlympiad == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("origin_olympiad", "body"))
	}
	if body.DifficultyRating == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("difficulty_rating", "body"))
	}
	if body.DifficultyRating != nil {
		if !(*body.DifficultyRating == 1 || *body.DifficultyRating == 2 || *body.DifficultyRating == 3 || *body.DifficultyRating == 4 || *body.DifficultyRating == 5) {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.difficulty_rating", *body.DifficultyRating, []any{1, 2, 3, 4, 5}))
		}
	}
	if body.DefaultMdStatement != nil {
		if err2 := ValidateMarkdownStatementResponse(body.DefaultMdStatement); err2 != nil {
			err = goa.MergeErrors(err, err2)
		}
	}
	for _, e := range body.Examples {
		if e != nil {
			if err2 := ValidateExampleResponse(e); err2 != nil {
				err = goa.MergeErrors(err, err2)
			}
		}
	}
	for _, e := range body.VisibleInputSubtasks {
		if e != nil {
			if err2 := ValidateStInputsResponse(e); err2 != nil {
				err = goa.MergeErrors(err, err2)
			}
		}
	}
	return
}

// ValidateMarkdownStatementResponse runs the validations defined on
// MarkdownStatementResponse
func ValidateMarkdownStatementResponse(body *MarkdownStatementResponse) (err error) {
	if body.Story == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("story", "body"))
	}
	if body.Input == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("input", "body"))
	}
	if body.Output == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("output", "body"))
	}
	return
}

// ValidateExampleResponse runs the validations defined on ExampleResponse
func ValidateExampleResponse(body *ExampleResponse) (err error) {
	if body.Input == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("input", "body"))
	}
	if body.Output == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("output", "body"))
	}
	return
}

// ValidateStInputsResponse runs the validations defined on StInputsResponse
func ValidateStInputsResponse(body *StInputsResponse) (err error) {
	if body.Subtask == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("subtask", "body"))
	}
	if body.Inputs == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("inputs", "body"))
	}
	return
}

// ValidateMarkdownStatementResponseBody runs the validations defined on
// MarkdownStatementResponseBody
func ValidateMarkdownStatementResponseBody(body *MarkdownStatementResponseBody) (err error) {
	if body.Story == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("story", "body"))
	}
	if body.Input == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("input", "body"))
	}
	if body.Output == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("output", "body"))
	}
	return
}

// ValidateExampleResponseBody runs the validations defined on
// ExampleResponseBody
func ValidateExampleResponseBody(body *ExampleResponseBody) (err error) {
	if body.Input == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("input", "body"))
	}
	if body.Output == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("output", "body"))
	}
	return
}

// ValidateStInputsResponseBody runs the validations defined on
// StInputsResponseBody
func ValidateStInputsResponseBody(body *StInputsResponseBody) (err error) {
	if body.Subtask == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("subtask", "body"))
	}
	if body.Inputs == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("inputs", "body"))
	}
	return
}
