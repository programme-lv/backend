package taskdomain

import (
	"fmt"

	"github.com/thoas/go-funk"
)

type Task struct {
	ShortId  string
	FullName string

	IllustrImgUrl string

	// constraints
	MemLimMegabytes int
	CpuTimeLimSecs  float64

	// metadata
	OriginOlympiad   string
	DifficultyRating int
	OriginNotes      []OriginNote

	// statement
	MdStatements   []MarkdownStatement
	PdfStatements  []PdfStatement
	VisInpSubtasks []VisibleInputSubtask
	Examples       []Example

	// evaluation
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
	LangIso639 string

	Story   string
	Input   string
	Output  string
	Notes   string
	Scoring string
	Talk    string // communication in interactive tasks
	Example string // example in interactive tasks

	Images []MdImgInfo
}

type MdImgInfo struct {
	Uuid  string
	S3Url string

	WidthPx  int
	HeightPx int
	WidthEm  int
}

type Subtask struct {
	Score   int
	TestIDs []int

	Descriptions map[string]string
}

type SubtaskWithId struct {
	ID int
	Subtask
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

// s3://proglv-tests/00a312d5348215f1afb97748059facead3a63babeb7ca24eea4eec012e8ee6bf.zst
func (test *Test) FullInputS3URI() string {
	format := "s3://proglv-tests/%s.zst"
	return fmt.Sprintf(format, test.InpSha2)
}

func (test *Test) FullAnswerS3URI() string {
	format := "s3://proglv-tests/%s.zst"
	return fmt.Sprintf(format, test.AnsSha2)
}

// https://proglv-tests.s3.eu-central-1.amazonaws.com/00a312d5348215f1afb97748059facead3a63babeb7ca24eea4eec012e8ee6bf.zst
func (test *Test) FullInputS3URL() string {
	format := "https://proglv-tests.s3.eu-central-1.amazonaws.com/%s.zst"
	return fmt.Sprintf(format, test.InpSha2)
}

func (test *Test) FullAnswerS3URL() string {
	format := "https://proglv-tests.s3.eu-central-1.amazonaws.com/%s.zst"
	return fmt.Sprintf(format, test.AnsSha2)
}

func (t *Task) FindSubtasksWithTest(testId int) []SubtaskWithId {
	subtasks := make([]SubtaskWithId, 0)
	for i, subtask := range t.Subtasks {
		for _, test := range subtask.TestIDs {
			if test == testId {
				subtasks = append(subtasks, SubtaskWithId{
					ID:      i + 1,
					Subtask: subtask,
				})
			}
		}
	}
	return subtasks
}

type TestGroupWithID struct {
	ID int
	TestGroup
}

func (t *Task) FindTestGroupsWithTest(testId int) []TestGroupWithID {
	testGroups := make([]TestGroupWithID, 0)
	for i, testGroup := range t.TestGroups {
		for _, test := range testGroup.TestIDs {
			if test == testId {
				testGroups = append(testGroups, TestGroupWithID{
					ID:        i + 1,
					TestGroup: testGroup,
				})
			}
		}
	}
	return testGroups
}

// TestWithOnlyInput represents a test with only its input data.
type TestWithOnlyInput struct {
	TestID int
	Input  string
}

// TestGroup represents a group of tests within a task.
type TestGroup struct {
	Points int
	Public bool
	// Subtask int
	TestIDs []int
}

func (t *Task) FindTestGroupSubtasks(testGroupId int) []int {
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

// ImgUuidUrl represents a mapping between image UUIDs and their URLs.
type ImgUuidUrl struct {
	UUID string
	Url  string
}

// OriginNote represents origin notes with language and information.
type OriginNote struct {
	Lang string
	Info string
}
