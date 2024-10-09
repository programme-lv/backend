package submsrvc

import (
	"time"

	"github.com/google/uuid"
)

type Submission struct {
	UUID uuid.UUID

	Content string

	Author Author
	Task   Task
	Lang   Lang

	CurrEval Evaluation

	CreatedAt time.Time
}

type Evaluation struct {
	UUID      uuid.UUID
	Stage     string
	CreatedAt time.Time

	ScoreBySubtasks   *SubtaskScoringRes
	ScoreByTestGroups *TestGroupScoringRes
	ScoreByTestSets   *TestSetScoringRes
}

type Author struct {
	UUID     uuid.UUID
	Username string
}

type Lang struct {
	ShortID  string
	Display  string
	MonacoID string
}

type Task struct {
	ShortID  string
	FullName string
}

type SubtaskScoringRes struct {
	SubtaskID     int
	SubtaskPoints int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

type TestGroupScoringRes struct {
	TestGroupID     int
	TestGroupPoints int
	AcceptedTests   int
	WrongTests      int
	UntestedTests   int
}

type TestSetScoringRes struct {
	Accepted int
	Wrong    int
	Untested int
}
