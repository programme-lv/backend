package submsrvc

import (
	"time"

	"github.com/google/uuid"
)

type Submission struct {
	UUID uuid.UUID

	Content string

	Author Author
	Task   TaskRef
	Lang   PrLang

	CurrEval Evaluation

	CreatedAt time.Time
}

const (
	ScoreUnitTest      = "test"
	ScoreUnitTestGroup = "group"
	ScoreUnitSubtask   = "subtask"
)

const (
	StageWaiting   = "waiting"
	StageCompiling = "compiling"
	StageTesting   = "testing"
	StageFinished  = "finished"
)

type Evaluation struct {
	UUID      uuid.UUID
	Stage     string
	ScoreUnit string

	Error *EvaluationError

	Subtasks []Subtask
	Groups   []TestGroup

	Tests []Test

	Checker    *string
	Interactor *string
	CpuLimMs   int
	MemLimKiB  int

	CreatedAt time.Time
}

type EvaluationError struct {
	Type    EvaluationErrorType
	Message *string
}

type EvaluationErrorType string

const (
	ErrorTypeCompilation EvaluationErrorType = "compilation"
	ErrorTypeInternal    EvaluationErrorType = "internal"
)

type Test struct {
	Ac  bool // accepted
	Wa  bool // wrong answer
	Tle bool // time limit exceeded
	Mle bool // memory limit exceeded
	Re  bool // runtime error
	Ig  bool // ignored

	Reached  bool // reached by tester
	Finished bool // has finished
}

type Author struct {
	UUID     uuid.UUID
	Username string
}

type PrLang struct {
	ShortID  string
	Display  string
	MonacoID string
}

type TaskRef struct {
	ShortID  string
	FullName string
}

type Subtask struct {
	Points      int
	Description string // short md spec
	StTests     []int  // subtask tests
}

type TestGroup struct {
	Points   int
	Subtasks []int
	TgTests  []int // test group tests
}
