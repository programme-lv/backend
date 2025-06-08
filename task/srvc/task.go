package srvc

import (
	"github.com/thoas/go-funk"
)

type Task struct {
	// url slug friendly identifier
	ShortId string

	// full name of the task (TODO: translateable)
	FullName string

	// illustration image
	IllustrImg IllustrationImage

	// constraints
	MemLimMegabytes int
	CpuTimeLimSecs  float64

	// metadata (origin, difficulty)
	OriginOlympiad   string
	OriginNotes      []OriginNote
	DifficultyRating int

	// statement
	MdStatements   []MarkdownStatement
	MdImages       []StatementImage
	PdfStatements  []PdfStatement
	VisInpSubtasks []VisibleInputSubtask
	Examples       []Example

	// testing
	Tests      []Test
	Checker    string
	Interactor string

	// scoring
	Subtasks   []Subtask
	TestGroups []TestGroup
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

type Example struct {
	Input  string
	Output string
	MdNote string
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
	WidthPx   *int
	HeightPx  *int
	SzInBytes *int
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
	ObjectUrl  string
}

// OriginNote represents origin notes with language and information.
type OriginNote struct {
	Lang string
	Info string
}
