package http

import "github.com/programme-lv/backend/tasksrvc"

type Example struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	MdNote string `json:"md_note,omitempty"`
}

type MdStatement struct {
	Story   string  `json:"story"`
	Input   string  `json:"input"`
	Output  string  `json:"output"`
	Notes   *string `json:"notes,omitempty"`
	Scoring *string `json:"scoring,omitempty"`
}

type StInputs struct {
	Subtask int      `json:"subtask"`
	Inputs  []string `json:"inputs"`
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
	VisibleInputSubtasks   []StInputs        `json:"visible_input_subtasks"`
}

func mapStInputs(stInputs []tasksrvc.StInputs) []StInputs {
	response := make([]StInputs, len(stInputs))
	for i, st := range stInputs {
		response[i] = StInputs{
			Subtask: st.Subtask,
			Inputs:  st.Inputs,
		}
	}
	return response
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
	response := &Task{
		PublishedTaskID:        task.PublishedTaskID,
		TaskFullName:           task.TaskFullName,
		MemoryLimitMegabytes:   task.MemoryLimitMegabytes,
		CPUTimeLimitSeconds:    task.CPUTimeLimitSeconds,
		OriginOlympiad:         task.OriginOlympiad,
		IllustrationImgURL:     task.IllustrationImgURL,
		DifficultyRating:       task.DifficultyRating,
		DefaultMDStatement:     mapTaskMdStatement(task.DefaultMdStatement),
		Examples:               mapTaskExamples(task.Examples),
		DefaultPDFStatementURL: task.DefaultPdfStatementURL,
		OriginNotes:            task.OriginNotes,
		VisibleInputSubtasks:   mapStInputs(task.VisibleInputSubtasks),
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
