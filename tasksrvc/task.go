package tasksrvc

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
	SubtaskID int
	Score     int
	TestIDs   []int
}

type TaskEvalTestGroupInformation struct {
	TestGroupID int
	Score       int
	Subtask     int
}

type TaskSubmEvalData struct {
	PublishedTaskID      string
	TaskFullName         string
	MemoryLimitMegabytes int
	CPUTimeLimitSeconds  float64
	Tests                []*Test
	TestlibCheckerCode   string
	TestGroups           []TestGroup
}

type Test struct {
	TestID          int
	FullInputS3URI  string
	InputSha256     string
	FullAnswerS3URI string
	AnswerSha256    string
	Subtasks        []int
	TestGroup       *int
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
	GroupID int
	Points  int
	Public  bool
	Subtask int
	TestIDs []int
}

// TestChecksum represents the checksums for a test's input and answer.
type TestChecksum struct {
	TestID  int
	InSHA2  string
	AnsSHA2 string
}

// PdfStatement represents a PDF statement with language and checksum.
type PdfStatement struct {
	LangIso639 string
	ObjectUrl  string
}

// ImgUuidS3Pair represents a mapping between image UUIDs and their S3 keys.
type ImgUuidS3Pair struct {
	UUID  string
	S3Key string
}

// OriginNote represents origin notes with language and information.
type OriginNote struct {
	Lang string
	Info string
}
