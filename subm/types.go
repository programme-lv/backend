package subm

type CreateSubmissionPayload struct {
	Submission        string
	Username          string
	ProgrammingLangID string
	TaskCodeID        string
	Token             string
}

type Submission struct {
	SubmUUID              string
	Submission            string
	Username              string
	CreatedAt             string
	EvalUUID              string
	EvalStatus            string
	EvalScoringTestgroups []*TestGroupResult
	EvalScoringTests      *TestsResult
	EvalScoringSubtasks   []*SubtaskResult
	PLangID               string
	PLangDisplayName      string
	PLangMonacoID         string
	TaskName              string
	TaskID                string
}

type SubmissionListUpdate struct {
	SubmCreated        *Submission
	StateUpdate        *SubmissionStateUpdate
	TestgroupResUpdate *TestgroupScoreUpdate
}

type SubtaskResult struct {
	SubtaskID     int
	SubtaskScore  int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

type TestGroupResult struct {
	TestGroupID      int
	TestGroupScore   int
	StatementSubtask int
	AcceptedTests    int
	WrongTests       int
	UntestedTests    int
}

type TestgroupScoreUpdate struct {
	SubmUUID      string
	EvalUUID      string
	TestGroupID   int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

type TestsResult struct {
	Accepted int
	Wrong    int
	Untested int
}
