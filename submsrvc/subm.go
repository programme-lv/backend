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

	Subtasks []Subtask
	Groups   []TestGroup

	TestSet *TestSet
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

type Subtask struct {
	SubtaskID   int
	Points      int
	Accepted    int
	Wrong       int
	Untested    int
	Description string
}

type TestGroup struct {
	GroupID  int
	Points   int
	Accepted int
	Wrong    int
	Untested int
	Subtasks []int
}

type TestSet struct {
	Accepted int
	Wrong    int
	Untested int
}
