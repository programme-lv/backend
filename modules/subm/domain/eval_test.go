package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

func TestAddExamplesAppendsUnscoredTests(t *testing.T) {
	task := tasksrvc.Task{
		Examples: []tasksrvc.Example{
			{Input: "1 2\n", Output: "3\n"},
			{Input: "4 5\n", Output: "9\n"},
		},
	}
	eval := Eval{ScoreUnit: ScoreUnitTest}
	eval.AddExamples(task)

	assert.Len(t, eval.Tests, 2)
	assert.True(t, eval.Tests[0].Example)
	assert.Equal(t, "1 2\n", eval.Tests[0].Input)
	assert.Equal(t, "3\n", eval.Tests[0].Answer)
}

func TestAddExamplesSkipsInteractiveTraces(t *testing.T) {
	task := tasksrvc.Task{
		Interactor: "interactor.cpp",
		Examples:   []tasksrvc.Example{{Trace: "1\n"}},
	}
	eval := Eval{ScoreUnit: ScoreUnitTest}
	eval.AddExamples(task)

	assert.Empty(t, eval.Tests)
}
