package tasksrvc

import "fmt"

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
	VisInpSubtasks []VisInpSubtask
	Examples       []Example

	// evaluation
	Tests      []Test
	Checker    string
	Interactor string

	// scoring
	Subtasks   []Subtask
	TestGroups []TestGroup
}

type Example struct {
	OrderId int
	Input   string
	Output  string
	MdNote  string
}

type MarkdownStatement struct {
	LangIso639 string

	Story   string
	Input   string
	Output  string
	Notes   string
	Scoring string
}

type Subtask struct {
	ID      int
	Score   int
	TestIDs []int
}

type TaskEvalTestGroupInformation struct {
	TestGroupID int
	Score       int
	Subtask     int
}

type Test struct {
	ID      int
	InpSha2 string
	AnsSha2 string
}

func (test *Test) FullInputS3URI() string {
	format := "s3://proglv-tests/%s.zst"
	return fmt.Sprintf(format, test.InpSha2)
}

func (test *Test) FullAnswerS3URI() string {
	format := "s3://proglv-tests/%s.zst"
	return fmt.Sprintf(format, test.AnsSha2)
}

func (t *Task) FindSubtasksWithTest(testId int) []Subtask {
	subtasks := make([]Subtask, 0)
	for _, subtask := range t.Subtasks {
		for _, test := range subtask.TestIDs {
			if test == testId {
				subtasks = append(subtasks, subtask)
			}
		}
	}
	return subtasks
}

func (t *Task) FindTestGroupsWithTest(testId int) []TestGroup {
	testGroups := make([]TestGroup, 0)
	for _, testGroup := range t.TestGroups {
		for _, test := range testGroup.TestIDs {
			if test == testId {
				testGroups = append(testGroups, testGroup)
			}
		}
	}
	return testGroups
}

// VisInpSubtask represents a subtask with visible input.
// Usually, such subtasks are used to gift students some points by
// allowing them solve some testcases by hand.
type VisInpSubtask struct {
	Subtask int
	Inputs  []TestWithOnlyInput
}

// TestWithOnlyInput represents a test with only its input data.
type TestWithOnlyInput struct {
	TestID int
	Input  string
}

// TestGroup represents a group of tests within a task.
type TestGroup struct {
	ID      int
	Points  int
	Public  bool
	Subtask int
	TestIDs []int
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
