package srvc

import (
	"github.com/thoas/go-funk"
)

type TaskPreview struct {
	ShortId string

	FullName map[string]string
	OrigLang string

	IllustrImg *IllustrationImage

	DifficultyRating int

	OriginOlympiad string
	OriginNote     string
	OriginOrg      string
	OriginYear     string
	OlympStage     string

	MdStatementStory string
}

type Task struct {
	// url slug friendly identifier
	ShortId string

	// full name of the task in multiple languages (key: ISO 639 code)
	FullName map[string]string

	// original language of the task (ISO 639 code), may be empty
	OrigLang string

	// markdown with todos, notes, etc.
	Readme string

	// illustration image
	IllustrImg *IllustrationImage

	// constraints
	MemLimMegabytes int
	CpuTimeLimSecs  float64

	// metadata (origin, difficulty)
	OriginOlympiad   string
	OriginNotes      []OriginNote
	DifficultyRating int
	OriginOrg        string
	OriginYear       string
	OlympStage       string

	// statement
	MdStatements   []MarkdownStatement
	MdImages       []StatementImage
	PdfStatements  []PdfStatement
	VisInpSubtasks []VisibleInputSubtask
	Examples       []Example
	Subtasks       []Subtask

	// testing
	Tests      []Test
	Checker    string
	Interactor string

	// scoring
	TestGroups []TestGroup

	// metadata: authors (free-form names)
	Authors []string

	// metadata: problem tags (free-form short labels)
	ProblemTags []string

	// original full archive S3 key (optional)
	ArchiveS3Key string
}

func (t *Task) CpuMillis() int {
	return int(t.CpuTimeLimSecs * 1000)
}

func (t *Task) MemoryKiB() int {
	// 1 MB = 976.5625 KiB
	return int(float64(t.MemLimMegabytes) * 976.5625)
}

func (t *Task) CheckerPtr() *string {
	if t.Checker != "" {
		return &t.Checker
	}
	return nil
}

func (t *Task) InteractorPtr() *string {
	if t.Interactor != "" {
		return &t.Interactor
	}
	return nil
}

// DefaultFullName returns the preferred full name, prioritizing OrigLang, then 'lv', then any available name.
func (t *Task) DefaultFullName() string {
	if t.FullName == nil {
		return ""
	}
	if t.OrigLang != "" {
		if v, ok := t.FullName[t.OrigLang]; ok {
			return v
		}
	}
	if v, ok := t.FullName["lv"]; ok {
		return v
	}
	for _, v := range t.FullName {
		return v
	}
	return ""
}

type Example struct {
	Input  string
	Output string
	MdNote map[string]string
}

type VisibleInputSubtask struct {
	SubtaskId int
	Tests     []VisInpSubtaskTest
}

type VisInpSubtaskTest struct {
	TestId int
	Input  string
}

type MarkdownStatement struct {
	LangIso639 string // primary key of the statement of task

	Story   string
	Input   string
	Output  string
	Notes   string
	Scoring string
	Talk    string // communication in interactive tasks
	Example string // example in interactive tasks
}

type IllustrationImage struct {
	S3Key     string
	WidthPx   int
	HeightPx  int
	SzInBytes int
}

type StatementImage struct {
	S3Key     string // e.g. task-md-images/<sanitized-filename>.png
	Filename  string // filename of the image, e.g., nekoks.png
	WidthPx   int    // og width [px] stored in s3
	HeightPx  int    // og height [px] stored in s3
	SzInBytes int    // size in bytes
}

type Subtask struct {
	Score   int
	TestIDs []int

	Descriptions map[string]string
}

type TaskEvalTestGroupInformation struct {
	TestGroupID int
	Score       int
	Subtask     int
}

type Test struct {
	InpSha2 string
	AnsSha2 string
}

// TestGroup represents a group of tests within a task.
type TestGroup struct {
	Points int
	Public bool
	// Subtask int
	TestIDs []int
}

func (t *Task) FindTestgroupSubtasks(testGroupId int) []int {
	tests := make([]int, 0)
	tests = append(tests, t.TestGroups[testGroupId-1].TestIDs...)

	subtasks := make([]int, 0)
	for i, subtask := range t.Subtasks {
		for _, test := range subtask.TestIDs {
			if funk.ContainsInt(tests, test) {
				subtasks = append(subtasks, i+1)
				break
			}
		}
	}
	return subtasks
}

// PdfStatement represents a PDF statement with language and checksum.
type PdfStatement struct {
	LangIso639 string
	S3Key      string
}

// OriginNote represents origin notes with language and information.
type OriginNote struct {
	Lang string
	Info string
}

// DefaultFullName returns the preferred full name for TaskPreview.
func (t *TaskPreview) DefaultFullName() string {
	if t.FullName == nil {
		return ""
	}
	if t.OrigLang != "" {
		if v, ok := t.FullName[t.OrigLang]; ok {
			return v
		}
	}
	if v, ok := t.FullName["lv"]; ok {
		return v
	}
	for _, v := range t.FullName {
		return v
	}
	return ""
}
