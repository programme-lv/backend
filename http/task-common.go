package http

import "github.com/programme-lv/backend/tasksrvc"

type Example struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	MdNote string `json:"md_note,omitempty"`
}

type MdStatement struct {
	Story   string `json:"story"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Notes   string `json:"notes,omitempty"`
	Scoring string `json:"scoring,omitempty"`
}

type VisInputSubtask struct {
	Subtask    int                 `json:"subtask"`
	TestInputs []TestWithOnlyInput `json:"inputs"`
}

type TestWithOnlyInput struct {
	TestId int    `json:"test_id"`
	Input  string `json:"input"`
}

type Task struct {
	PublishedTaskID        string            `json:"published_task_id"`
	TaskFullName           string            `json:"task_full_name"`
	MemoryLimitMegabytes   int               `json:"memory_limit_megabytes"`
	CPUTimeLimitSeconds    float64           `json:"cpu_time_limit_seconds"`
	OriginOlympiad         string            `json:"origin_olympiad"`
	IllustrationImgURL     *string           `json:"illustration_img_url"`
	DifficultyRating       *int              `json:"difficulty_rating"`
	DefaultMDStatement     MdStatement       `json:"default_md_statement"`
	Examples               []Example         `json:"examples"`
	DefaultPDFStatementURL *string           `json:"default_pdf_statement_url"`
	OriginNotes            map[string]string `json:"origin_notes"`
	VisibleInputSubtasks   []VisInputSubtask `json:"visible_input_subtasks"`
}

func mapTaskMdStatement(md *tasksrvc.MarkdownStatement) MdStatement {
	if md == nil {
		return MdStatement{}
	}
	return MdStatement{
		Story:   md.Story,
		Input:   md.Input,
		Output:  md.Output,
		Notes:   md.Notes,
		Scoring: md.Scoring,
	}
}

func mapTaskExamples(examples []tasksrvc.Example) []Example {
	response := make([]Example, len(examples))
	for i, e := range examples {
		response[i] = Example{
			Input:  e.Input,
			Output: e.Output,
			MdNote: e.MdNote,
		}
	}
	return response
}

func mapTaskResponse(task *tasksrvc.Task) *Task {
	illstrImgUrl := new(string)
	if task.IllustrImgUrl != "" {
		illstrImgUrl = new(string)
		*illstrImgUrl = task.IllustrImgUrl
	}

	difficultyRating := new(int)
	if task.DifficultyRating != 0 {
		difficultyRating = new(int)
		*difficultyRating = task.DifficultyRating
	}

	response := &Task{
		PublishedTaskID:        task.ShortId,
		TaskFullName:           task.FullName,
		MemoryLimitMegabytes:   task.MemLimMegabytes,
		CPUTimeLimitSeconds:    task.CpuTimeLimSecs,
		OriginOlympiad:         task.OriginOlympiad,
		IllustrationImgURL:     illstrImgUrl,
		DifficultyRating:       difficultyRating,
		DefaultMDStatement:     mapTaskMdStatement(nil),
		Examples:               mapTaskExamples(task.Examples),
		DefaultPDFStatementURL: nil,
		OriginNotes:            nil,
		VisibleInputSubtasks:   []VisInputSubtask{}, // TODO: add visible input subtasks
	}
	return response
}

func mapTasksResponse(tasks []*tasksrvc.Task) []*Task {
	response := make([]*Task, len(tasks))
	for i, task := range tasks {
		response[i] = mapTaskResponse(task)
	}
	return response
}
