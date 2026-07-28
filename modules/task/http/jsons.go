package http

import (
	"context"
	"log/slog"

	"github.com/programme-lv/backend/modules/task/srvc"
)

type Empty struct{}

type TaskPreview struct {
	ShortId          string             `json:"short_id"`
	FullName         string             `json:"full_name"`
	IllustrImg       *IllustrationImage `json:"illustr_img"`
	DifficultyRating int                `json:"difficulty_rating"`
	OriginOlympiad   string             `json:"origin_olympiad"`
	OriginNote       string             `json:"origin_note"`
	OriginNoteShort  string             `json:"origin_note_short"`
	MdStatementStory string             `json:"md_statement_story"`
}

type Example struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	MdNote string `json:"md_note,omitempty"`
}

type MdStatement struct {
	Story   string `json:"story"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Notes   string `json:"notes"`
	Scoring string `json:"scoring"`
	Talk    string `json:"talk"`
	Example string `json:"example"`
	// Images  []MdImg `json:"images"`
}

type VisInputSubtask struct {
	SubtaskID  int                 `json:"subtask"`
	TestInputs []TestWithOnlyInput `json:"inputs"`
}

type TestWithOnlyInput struct {
	TestID int    `json:"test_id"`
	Input  string `json:"input"`
}

type StatementImage struct {
	S3Key     string `json:"s3_key"`
	Filename  string `json:"filename"`
	HttpUrl   string `json:"http_url"`
	WidthPx   int    `json:"width_px"`
	HeightPx  int    `json:"height_px"`
	SzInBytes int    `json:"sz_in_bytes"`
}

type IllustrationImage struct {
	HttpUrl   string `json:"http_url"`
	WidthPx   int    `json:"width_px"`
	HeightPx  int    `json:"height_px"`
	SzInBytes int    `json:"sz_in_bytes"`
}

type Task struct {
	ShortTaskID            string             `json:"short_task_id"`
	TaskFullName           string             `json:"task_full_name"`
	MemoryLimitMegabytes   int                `json:"memory_limit_megabytes"`
	CPUTimeLimitSeconds    float64            `json:"cpu_time_limit_seconds"`
	OriginOlympiad         string             `json:"origin_olympiad"`
	IllustrationImg        *IllustrationImage `json:"illustration_img"`
	DifficultyRating       *int               `json:"difficulty_rating"`
	DefaultMDStatement     MdStatement        `json:"default_md_statement"`
	StatementImages        []StatementImage   `json:"statement_images"`
	Examples               []Example          `json:"examples"`
	DefaultPDFStatementURL *string            `json:"default_pdf_statement_url"`
	OriginNotes            map[string]string  `json:"origin_notes"`
	VisibleInputSubtasks   []VisInputSubtask  `json:"visible_input_subtasks"`
	StatementSubtasks      []SubtaskOverview  `json:"statement_subtasks"`
	TestingType            string             `json:"testing_type"`
}

type SubtaskOverview struct {
	SubtaskID    int               `json:"subtask"`
	Score        int               `json:"score"`
	Descriptions map[string]string `json:"descriptions"`
}

func mapTaskMdStatement(md *srvc.MarkdownStatement) MdStatement {
	if md == nil {
		return MdStatement{}
	}
	return MdStatement{
		Story:   md.Story,
		Input:   md.Input,
		Output:  md.Output,
		Notes:   md.Notes,
		Scoring: md.Scoring,
		Talk:    md.Talk,
		Example: md.Example,
		// Images: imgSizes,
	}
}

func mapTaskExamples(examples []srvc.Example) []Example {
	response := make([]Example, len(examples))
	for i, e := range examples {
		mdNote := ""
		if e.MdNote != nil {
			if v, ok := e.MdNote["lv"]; ok {
				mdNote = v
			} else if v, ok := e.MdNote["en"]; ok {
				mdNote = v
			} else {
				for _, v := range e.MdNote {
					mdNote = v
					break
				}
			}
		}
		response[i] = Example{
			Input:  e.Input,
			Output: e.Output,
			MdNote: mdNote,
		}
	}
	return response
}

func (h *taskHttpHandler) mapTaskResponse(task srvc.Task) Task {
	difficultyRating := new(int)
	if task.DifficultyRating != 0 {
		difficultyRating = new(int)
		*difficultyRating = task.DifficultyRating
	}

	pdfStatements := task.PdfStatements
	defaultPdfStatementUrl := new(string)
	for _, pdfStatement := range pdfStatements {
		if pdfStatement.LangIso639 == "lv" {
			url, err := h.taskSrvc.GetHttpUrlForPdfStatement(context.TODO(), pdfStatement.S3Key)
			if err != nil {
				slog.Error("failed to get public url for pdf statement", "error", err)
				url = ""
			}
			*defaultPdfStatementUrl = url
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

	testingType := "checker"
	if task.Interactor != "" {
		testingType = "interactor"
	}

	response := Task{
		ShortTaskID:            task.ShortId,
		TaskFullName:           task.DefaultFullName(),
		MemoryLimitMegabytes:   task.MemLimMegabytes,
		CPUTimeLimitSeconds:    task.CpuTimeLimSecs,
		OriginOlympiad:         task.OriginOlympiad,
		IllustrationImg:        h.mapTaskIllustrImg(task.IllustrImg),
		DifficultyRating:       difficultyRating,
		DefaultMDStatement:     defaultMdStatement,
		StatementImages:        h.mapTaskStatementImages(task.MdImages),
		Examples:               mapTaskExamples(task.Examples),
		DefaultPDFStatementURL: defaultPdfStatementUrl,
		OriginNotes:            originNotesAsAMap,
		VisibleInputSubtasks:   visInputSubtasks,
		StatementSubtasks:      subtasks,
		TestingType:            testingType,
	}
	return response
}

func (h *taskHttpHandler) mapTaskIllustrImg(illustrImg *srvc.IllustrationImage) *IllustrationImage {
	if illustrImg == nil || illustrImg.S3Key == "" {
		return nil
	}

	httpUrl, err := h.taskSrvc.GetHttpUrlForIllustrImg(context.TODO(), illustrImg.S3Key)
	if err != nil {
		slog.Error("failed to get public url for illustration image", "error", err)
		return nil
	}

	return &IllustrationImage{
		HttpUrl:   httpUrl,
		WidthPx:   illustrImg.WidthPx,
		HeightPx:  illustrImg.HeightPx,
		SzInBytes: illustrImg.SzInBytes,
	}
}

func (h *taskHttpHandler) mapTaskPreview(preview srvc.TaskPreview) TaskPreview {
	return TaskPreview{
		ShortId:          preview.ShortId,
		FullName:         preview.DefaultFullName(),
		IllustrImg:       h.mapTaskIllustrImg(preview.IllustrImg),
		DifficultyRating: preview.DifficultyRating,
		OriginOlympiad:   preview.OriginOlympiad,
		OriginNote:       preview.OriginNote,
		OriginNoteShort:  preview.OriginNoteShort,
		MdStatementStory: preview.MdStatementStory,
	}
}

func (h *taskHttpHandler) mapTaskStatementImages(images []srvc.StatementImage) []StatementImage {
	response := make([]StatementImage, len(images))
	for i, image := range images {
		httpUrl, err := h.taskSrvc.GetHttpUrlForStatementImage(context.TODO(), image.S3Key)
		if err != nil {
			slog.Error("failed to get public url for statement image", "error", err)
			httpUrl = ""
		}
		response[i] = StatementImage{
			S3Key:     image.S3Key,
			Filename:  image.Filename,
			HttpUrl:   httpUrl,
			WidthPx:   image.WidthPx,
			HeightPx:  image.HeightPx,
			SzInBytes: image.SzInBytes,
		}
	}
	return response
}
