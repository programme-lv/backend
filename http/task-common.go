package http

import (
	"strings"

	"github.com/programme-lv/backend/task/taskdomain"
)

type Example struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	MdNote string `json:"md_note,omitempty"`
}

type MdStatement struct {
	Story   string  `json:"story"`
	Input   string  `json:"input"`
	Output  string  `json:"output"`
	Notes   string  `json:"notes"`
	Scoring string  `json:"scoring"`
	Talk    string  `json:"talk"`
	Example string  `json:"example"`
	Images  []MdImg `json:"images"`
}

type MdImg struct {
	ImgUuid  string `json:"img_uuid"`
	HttpUrl  string `json:"http_url"`
	WidthEm  int    `json:"width_em"`
	WidthPx  int    `json:"width_px"`
	HeightPx int    `json:"height_px"`
}

type VisInputSubtask struct {
	SubtaskID  int                 `json:"subtask"`
	TestInputs []TestWithOnlyInput `json:"inputs"`
}

type TestWithOnlyInput struct {
	TestID int    `json:"test_id"`
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
	StatementSubtasks      []SubtaskOverview `json:"statement_subtasks"`
}

type SubtaskOverview struct {
	SubtaskID    int               `json:"subtask"`
	Score        int               `json:"score"`
	Descriptions map[string]string `json:"descriptions"`
}

const PublicCloudfrontEndpoint = "https://dvhk4hiwp1rmf.cloudfront.net/"

func mapTaskMdStatement(md *taskdomain.MarkdownStatement) MdStatement {
	if md == nil {
		return MdStatement{}
	}
	imgSizes := make([]MdImg, len(md.Images))
	for i, img := range md.Images {
		oldPrefix := "https://proglv-public.s3.eu-central-1.amazonaws.com/"
		newPrefix := PublicCloudfrontEndpoint
		httpUrl := strings.Replace(img.S3Url, oldPrefix, newPrefix, 1)
		imgSizes[i] = MdImg{
			ImgUuid:  img.Uuid,
			HttpUrl:  httpUrl,
			WidthEm:  img.WidthEm,
			WidthPx:  img.WidthPx,
			HeightPx: img.HeightPx,
		}
	}
	return MdStatement{
		Story:   md.Story,
		Input:   md.Input,
		Output:  md.Output,
		Notes:   md.Notes,
		Scoring: md.Scoring,
		Talk:    md.Talk,
		Example: md.Example,

		Images: imgSizes,
	}
}

func mapTaskExamples(examples []taskdomain.Example) []Example {
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

func mapTaskResponse(task *taskdomain.Task) *Task {
	illstrImgUrl := new(string)
	if task.IllustrImgUrl != "" {
		illstrImgUrl = new(string)
		*illstrImgUrl = task.IllustrImgUrl

		*illstrImgUrl = strings.Replace(*illstrImgUrl, "https://proglv-public.s3.eu-central-1.amazonaws.com/", PublicCloudfrontEndpoint, 1)
	}

	difficultyRating := new(int)
	if task.DifficultyRating != 0 {
		difficultyRating = new(int)
		*difficultyRating = task.DifficultyRating
	}

	pdfStatements := task.PdfStatements
	defaultPdfStatementUrl := new(string)
	for _, pdfStatement := range pdfStatements {
		if pdfStatement.LangIso639 == "lv" {
			*defaultPdfStatementUrl = pdfStatement.ObjectUrl
		}
	}

	originNotes := task.OriginNotes
	originNotesAsAMap := make(map[string]string)
	for _, originNote := range originNotes {
		originNotesAsAMap[originNote.Lang] = originNote.Info
	}

	mdStatements := task.MdStatements
	defaultMdStatement := MdStatement{}
	foundMd := false
	// check if there is an lv statement
	for _, mdStatement := range mdStatements {
		if mdStatement.LangIso639 == "lv" {
			defaultMdStatement = mapTaskMdStatement(&mdStatement)
			foundMd = true
			break
		}
	}
	// if there is no lv statement, check if there is an en statement
	if !foundMd {
		for _, mdStatement := range mdStatements {
			if mdStatement.LangIso639 == "en" {
				defaultMdStatement = mapTaskMdStatement(&mdStatement)
				foundMd = true
				break
			}
		}
	}
	// if there is no en statement, pick the first statement
	if !foundMd {
		defaultMdStatement = mapTaskMdStatement(&mdStatements[0])
	}

	visInputSubtasks := make([]VisInputSubtask, 0)
	for _, visInputSt := range task.VisInpSubtasks {
		testInputs := make([]TestWithOnlyInput, 0)
		for _, test := range visInputSt.Tests {
			testInputs = append(testInputs, TestWithOnlyInput{
				TestID: test.TestId,
				Input:  test.Input,
			})
		}
		visInputSubtasks = append(visInputSubtasks, VisInputSubtask{
			SubtaskID:  visInputSt.SubtaskId,
			TestInputs: testInputs,
		})
	}

	subtasks := make([]SubtaskOverview, 0)
	for i, subtask := range task.Subtasks {
		subtasks = append(subtasks, SubtaskOverview{
			SubtaskID:    i + 1,
			Score:        subtask.Score,
			Descriptions: subtask.Descriptions,
		})
	}

	response := &Task{
		PublishedTaskID:        task.ShortId,
		TaskFullName:           task.FullName,
		MemoryLimitMegabytes:   task.MemLimMegabytes,
		CPUTimeLimitSeconds:    task.CpuTimeLimSecs,
		OriginOlympiad:         task.OriginOlympiad,
		IllustrationImgURL:     illstrImgUrl,
		DifficultyRating:       difficultyRating,
		DefaultMDStatement:     defaultMdStatement,
		Examples:               mapTaskExamples(task.Examples),
		DefaultPDFStatementURL: defaultPdfStatementUrl,
		OriginNotes:            originNotesAsAMap,
		VisibleInputSubtasks:   visInputSubtasks,
		StatementSubtasks:      subtasks,
	}
	return response
}

func mapTasksResponse(tasks []taskdomain.Task) []*Task {
	response := make([]*Task, len(tasks))
	for i, task := range tasks {
		response[i] = mapTaskResponse(&task)
	}
	return response
}
