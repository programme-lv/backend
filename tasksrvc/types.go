package tasksrvc

type Example struct {
	ExampleID int
	Input     string
	Output    string
	MdNote    *string
}

type GetTaskSubmEvalDataPayload struct {
	TaskID string
}

type MarkdownStatement struct {
	LangISO639 string
	Story      string
	Input      string
	Output     string
	Notes      *string
	Scoring    *string
}

type StInputs struct {
	Subtask int
	Inputs  []string
}

type Task struct {
	PublishedTaskID        string
	TaskFullName           string
	MemoryLimitMegabytes   int
	CPUTimeLimitSeconds    float64
	OriginOlympiad         string
	IllustrationImgURL     *string
	DifficultyRating       int
	DefaultMdStatement     *MarkdownStatement
	Examples               []*Example
	DefaultPdfStatementURL *string
	OriginNotes            map[string]string
	VisibleInputSubtasks   []*StInputs
}

type TaskEvalSubtaskScore struct {
	SubtaskID int
	Score     int
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
	Tests                []*TaskEvalTestInformation
	TestlibCheckerCode   string
	SubtaskScores        []*TaskEvalSubtaskScore
	TestGroupInformation []*TaskEvalTestGroupInformation
}

type TaskEvalTestInformation struct {
	TestID          int
	FullInputS3URI  string
	InputSha256     string
	FullAnswerS3URI string
	AnswerSha256    string
	Subtasks        []int
	TestGroup       *int
}

type VisInpSt struct {
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
	LangISO639 string
	PdfSHA2    string
}

// ImageUUIDMap represents a mapping between image UUIDs and their S3 keys.
type ImageUUIDMap struct {
	UUID  string
	S3Key string
}

// OriginNote represents origin notes with language and information.
type OriginNote struct {
	LangISO639 string
	OgInfo     string
}

// CreatePublicTaskInput encapsulates all data required to create a public task.
type CreatePublicTaskInput struct {
	TaskCode    string
	FullName    string  // Full name of the task
	MemMBytes   int     // Max memory usage during execution in megabytes
	CpuSecs     float64 // Max execution CPU time in seconds
	Difficulty  *int    // Integer from 1 to 5. 1 - very easy, 5 - very hard
	OriginOlymp string  // Name of the Olympiad where the task was used
	IllustrKey  *string // S3 key for bucket "proglv-public"
	VisInpSts   []VisInpSt
	TestGroups  []TestGroup
	TestChsums  []TestChecksum
	PdfSttments []PdfStatement
	MdSttments  []MarkdownStatement
	ImgUuidMap  []ImageUUIDMap
	Examples    []Example
	OriginNotes []OriginNote
}