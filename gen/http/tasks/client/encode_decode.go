// Code generated by goa v3.18.2, DO NOT EDIT.
//
// tasks HTTP client encoders and decoders
//
// Command:
// $ goa gen github.com/programme-lv/backend/design

package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	tasks "github.com/programme-lv/backend/gen/tasks"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

// BuildListTasksRequest instantiates a HTTP request object with method and
// path set to call the "tasks" service "listTasks" endpoint
func (c *Client) BuildListTasksRequest(ctx context.Context, v any) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: ListTasksTasksPath()}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("tasks", "listTasks", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeListTasksResponse returns a decoder for responses returned by the
// tasks listTasks endpoint. restoreBody controls whether the response body
// should be restored after having been read.
func DecodeListTasksResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body ListTasksResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("tasks", "listTasks", err)
			}
			for _, e := range body {
				if e != nil {
					if err2 := ValidateTaskResponse(e); err2 != nil {
						err = goa.MergeErrors(err, err2)
					}
				}
			}
			if err != nil {
				return nil, goahttp.ErrValidationError("tasks", "listTasks", err)
			}
			res := NewListTasksTaskOK(body)
			return res, nil
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("tasks", "listTasks", resp.StatusCode, string(body))
		}
	}
}

// BuildGetTaskRequest instantiates a HTTP request object with method and path
// set to call the "tasks" service "getTask" endpoint
func (c *Client) BuildGetTaskRequest(ctx context.Context, v any) (*http.Request, error) {
	var (
		taskID string
	)
	{
		p, ok := v.(*tasks.GetTaskPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("tasks", "getTask", "*tasks.GetTaskPayload", v)
		}
		taskID = p.TaskID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: GetTaskTasksPath(taskID)}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("tasks", "getTask", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeGetTaskResponse returns a decoder for responses returned by the tasks
// getTask endpoint. restoreBody controls whether the response body should be
// restored after having been read.
func DecodeGetTaskResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body GetTaskResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("tasks", "getTask", err)
			}
			err = ValidateGetTaskResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("tasks", "getTask", err)
			}
			res := NewGetTaskTaskOK(&body)
			return res, nil
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("tasks", "getTask", resp.StatusCode, string(body))
		}
	}
}

// unmarshalTaskResponseToTasksTask builds a value of type *tasks.Task from a
// value of type *TaskResponse.
func unmarshalTaskResponseToTasksTask(v *TaskResponse) *tasks.Task {
	res := &tasks.Task{
		PublishedTaskID:        *v.PublishedTaskID,
		TaskFullName:           *v.TaskFullName,
		MemoryLimitMegabytes:   *v.MemoryLimitMegabytes,
		CPUTimeLimitSeconds:    *v.CPUTimeLimitSeconds,
		OriginOlympiad:         *v.OriginOlympiad,
		IllustrationImgURL:     v.IllustrationImgURL,
		DifficultyRating:       *v.DifficultyRating,
		DefaultPdfStatementURL: v.DefaultPdfStatementURL,
	}
	if v.DefaultMdStatement != nil {
		res.DefaultMdStatement = unmarshalMarkdownStatementResponseToTasksMarkdownStatement(v.DefaultMdStatement)
	}
	if v.Examples != nil {
		res.Examples = make([]*tasks.Example, len(v.Examples))
		for i, val := range v.Examples {
			res.Examples[i] = unmarshalExampleResponseToTasksExample(val)
		}
	}
	if v.OriginNotes != nil {
		res.OriginNotes = make(map[string]string, len(v.OriginNotes))
		for key, val := range v.OriginNotes {
			tk := key
			tv := val
			res.OriginNotes[tk] = tv
		}
	}
	if v.VisibleInputSubtasks != nil {
		res.VisibleInputSubtasks = make([]*tasks.StInputs, len(v.VisibleInputSubtasks))
		for i, val := range v.VisibleInputSubtasks {
			res.VisibleInputSubtasks[i] = unmarshalStInputsResponseToTasksStInputs(val)
		}
	}

	return res
}

// unmarshalMarkdownStatementResponseToTasksMarkdownStatement builds a value of
// type *tasks.MarkdownStatement from a value of type
// *MarkdownStatementResponse.
func unmarshalMarkdownStatementResponseToTasksMarkdownStatement(v *MarkdownStatementResponse) *tasks.MarkdownStatement {
	if v == nil {
		return nil
	}
	res := &tasks.MarkdownStatement{
		Story:   *v.Story,
		Input:   *v.Input,
		Output:  *v.Output,
		Notes:   v.Notes,
		Scoring: v.Scoring,
	}

	return res
}

// unmarshalExampleResponseToTasksExample builds a value of type *tasks.Example
// from a value of type *ExampleResponse.
func unmarshalExampleResponseToTasksExample(v *ExampleResponse) *tasks.Example {
	if v == nil {
		return nil
	}
	res := &tasks.Example{
		Input:  *v.Input,
		Output: *v.Output,
		MdNote: v.MdNote,
	}

	return res
}

// unmarshalStInputsResponseToTasksStInputs builds a value of type
// *tasks.StInputs from a value of type *StInputsResponse.
func unmarshalStInputsResponseToTasksStInputs(v *StInputsResponse) *tasks.StInputs {
	if v == nil {
		return nil
	}
	res := &tasks.StInputs{
		Subtask: *v.Subtask,
	}
	res.Inputs = make([]string, len(v.Inputs))
	for i, val := range v.Inputs {
		res.Inputs[i] = val
	}

	return res
}

// unmarshalMarkdownStatementResponseBodyToTasksMarkdownStatement builds a
// value of type *tasks.MarkdownStatement from a value of type
// *MarkdownStatementResponseBody.
func unmarshalMarkdownStatementResponseBodyToTasksMarkdownStatement(v *MarkdownStatementResponseBody) *tasks.MarkdownStatement {
	if v == nil {
		return nil
	}
	res := &tasks.MarkdownStatement{
		Story:   *v.Story,
		Input:   *v.Input,
		Output:  *v.Output,
		Notes:   v.Notes,
		Scoring: v.Scoring,
	}

	return res
}

// unmarshalExampleResponseBodyToTasksExample builds a value of type
// *tasks.Example from a value of type *ExampleResponseBody.
func unmarshalExampleResponseBodyToTasksExample(v *ExampleResponseBody) *tasks.Example {
	if v == nil {
		return nil
	}
	res := &tasks.Example{
		Input:  *v.Input,
		Output: *v.Output,
		MdNote: v.MdNote,
	}

	return res
}

// unmarshalStInputsResponseBodyToTasksStInputs builds a value of type
// *tasks.StInputs from a value of type *StInputsResponseBody.
func unmarshalStInputsResponseBodyToTasksStInputs(v *StInputsResponseBody) *tasks.StInputs {
	if v == nil {
		return nil
	}
	res := &tasks.StInputs{
		Subtask: *v.Subtask,
	}
	res.Inputs = make([]string, len(v.Inputs))
	for i, val := range v.Inputs {
		res.Inputs[i] = val
	}

	return res
}