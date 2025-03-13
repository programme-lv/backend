package submdomain

import (
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/task/taskdomain"
)

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

	InpSha256 string
	AnsSha256 string
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

func NewEval(uuid uuid.UUID, submUuid uuid.UUID, task taskdomain.Task) Eval {
	subtasks := []Subtask{}
	for _, subtask := range task.Subtasks {
		subtasks = append(subtasks, Subtask{
			Points:      subtask.Score,
			Description: subtask.Descriptions["lv"],
			StTests:     subtask.TestIDs,
		})
	}

	testgroups := []TestGroup{}
	for i, tg := range task.TestGroups {
		testgroups = append(testgroups, TestGroup{
			Points:   tg.Points,
			Subtasks: task.FindTestGroupSubtasks(i + 1),
			TgTests:  tg.TestIDs,
		})
	}

	tests := []Test{}
	for _, test := range task.Tests {
		tests = append(tests, Test{
			Ac:        false,
			Wa:        false,
			Tle:       false,
			Mle:       false,
			Re:        false,
			Ig:        false,
			Reached:   false,
			Finished:  false,
			InpSha256: test.InpSha2,
			AnsSha256: test.AnsSha2,
		})
	}

	scoreUnit := ScoreUnitTest
	if len(task.Subtasks) > 0 {
		scoreUnit = ScoreUnitSubtask
	}
	if len(task.TestGroups) > 0 {
		scoreUnit = ScoreUnitTestGroup
	}

	return Eval{
		UUID:       uuid,
		SubmUUID:   submUuid,
		Stage:      EvalStageWaiting,
		ScoreUnit:  scoreUnit,
		Error:      nil,
		Subtasks:   subtasks,
		Groups:     testgroups,
		Tests:      tests,
		Checker:    task.CheckerPtr(),
		Interactor: task.InteractorPtr(),
		CpuLimMs:   task.CpuMillis(),
		MemLimKiB:  task.MemoryKiB(),
		CreatedAt:  time.Now(),
	}
}
