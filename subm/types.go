package subm

import (
	"time"

	"github.com/google/uuid"
)

type Subm struct {
	UUID         uuid.UUID
	Content      string
	AuthorUUID   uuid.UUID
	TaskShortID  string
	LangShortID  string
	CurrEvalUUID uuid.UUID
	CreatedAt    time.Time
}

type ScoreUnit string

const (
	ScoreUnitTest      ScoreUnit = "test"
	ScoreUnitTestGroup ScoreUnit = "group"
	ScoreUnitSubtask   ScoreUnit = "subtask"
)

type EvalStage string

const (
	EvalStageWaiting   EvalStage = "waiting"
	EvalStageCompiling EvalStage = "compiling"
	EvalStageTesting   EvalStage = "testing"
	EvalStageFinished  EvalStage = "finished"
)

type Eval struct {
	UUID      uuid.UUID
	SubmUUID  uuid.UUID
	Stage     EvalStage
	ScoreUnit ScoreUnit

	Error *EvalError

	Subtasks []Subtask
	Groups   []TestGroup

	Tests []Test

	Checker    *string
	Interactor *string
	CpuLimMs   int
	MemLimKiB  int

	CreatedAt time.Time
}

type EvalError struct {
	Type    EvalErrorType
	Message *string
}

type EvalErrorType string

const (
	ErrorTypeCompilation EvalErrorType = "compilation"
	ErrorTypeInternal    EvalErrorType = "internal"
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
