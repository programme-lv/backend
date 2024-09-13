package tasksrvc

type Example struct {
	Input  string
	Output string
	MdNote *string
}

type GetTaskSubmEvalDataPayload struct {
	TaskID string
}

type MarkdownStatement struct {
	Story   string
	Input   string
	Output  string
	Notes   *string
	Scoring *string
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
