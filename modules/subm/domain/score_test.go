package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateScoreSkipsExamples(t *testing.T) {
	cpu, mem := 10, 1024
	eval := Eval{
		ScoreUnit: ScoreUnitTest,
		Tests: []Test{
			{Ac: true, Reached: true, Finished: true, CpuMs: &cpu, MemKiB: &mem},
			{Example: true, Wa: true, Reached: true, Finished: true, CpuMs: &cpu, MemKiB: &mem},
		},
	}

	score := eval.CalculateScore()

	assert.Equal(t, 1, score.PossibleScore)
	assert.Equal(t, 1, score.ReceivedScore)
	assert.Equal(t, 100, score.ScoreBar.Green)
	assert.Equal(t, 0, score.ScoreBar.Red)
}

func TestCalculateScorePartialTestGroup(t *testing.T) {
	interactor := "interactor.cpp"
	eval := Eval{
		ScoreUnit:  ScoreUnitTestGroup,
		Interactor: &interactor,
		Groups: []TestGroup{
			{Points: 6, TgTests: []int{1, 2, 3}},
			{Points: 4, TgTests: []int{4, 5}},
		},
		Tests: []Test{
			{Ac: true, Reached: true, Finished: true},
			{Score: 40, Reached: true, Finished: true},
			{Wa: true, Reached: true, Finished: true},
			{Score: 40, Reached: true, Finished: true},
			{Score: 40, Reached: true, Finished: true},
		},
	}

	score := eval.CalculateScore()

	assert.Equal(t, 10, score.PossibleScore)
	assert.Equal(t, 44, score.ReceivedScoreTenths)
	assert.Equal(t, 4, score.ReceivedScore)
	assert.Equal(t, 100, score.ScoreBar.Yellow)
}

func TestCalculateScorePartialTestUnit(t *testing.T) {
	eval := Eval{
		ScoreUnit: ScoreUnitTest,
		Tests: []Test{
			{Score: 40, Reached: true, Finished: true},
			{Ac: true, Reached: true, Finished: true},
		},
	}

	score := eval.CalculateScore()

	assert.Equal(t, 2, score.PossibleScore)
	assert.Equal(t, 14, score.ReceivedScoreTenths)
	assert.Equal(t, 1, score.ReceivedScore)
	assert.Equal(t, 50, score.ScoreBar.Yellow)
	assert.Equal(t, 50, score.ScoreBar.Green)
}

func TestCalculateScoreSimpleGroupIsAllOrNothing(t *testing.T) {
	eval := Eval{
		ScoreUnit: ScoreUnitTestGroup,
		Groups:    []TestGroup{{Points: 6, TgTests: []int{1, 2, 3}}},
		Tests: []Test{
			{Ac: true, Reached: true, Finished: true},
			{Ac: true, Reached: true, Finished: true},
			{Wa: true, Reached: true, Finished: true},
		},
	}

	score := eval.CalculateScore()

	assert.Equal(t, 0, score.ReceivedScore)
	assert.Equal(t, 0, score.ReceivedScoreTenths)
	assert.Equal(t, 6, score.PossibleScore)
}

func TestCalculateScoreGroupBeingTestedIsYellow(t *testing.T) {
	eval := Eval{
		ScoreUnit: ScoreUnitTestGroup,
		Groups:    []TestGroup{{Points: 6, TgTests: []int{1, 2, 3}}},
		Tests: []Test{
			{Ac: true, Reached: true, Finished: true},
			{Reached: true},
			{},
		},
	}

	score := eval.CalculateScore()

	assert.Equal(t, 100, score.ScoreBar.Yellow)
	assert.Equal(t, 0, score.ScoreBar.Red)
	assert.Equal(t, 0, score.ScoreBar.Green)
}

func TestCalculateScoreGroupFailedWithIgnoredTestsIsRed(t *testing.T) {
	eval := Eval{
		ScoreUnit: ScoreUnitTestGroup,
		Groups:    []TestGroup{{Points: 6, TgTests: []int{1, 2, 3}}},
		Tests: []Test{
			{Wa: true, Reached: true, Finished: true},
			{Ig: true},
			{Ig: true},
		},
	}

	score := eval.CalculateScore()

	assert.Equal(t, 100, score.ScoreBar.Red)
	assert.Equal(t, 0, score.ScoreBar.Yellow)
}

func TestCalculateScoreSubtaskBeingTestedIsYellow(t *testing.T) {
	eval := Eval{
		ScoreUnit: ScoreUnitSubtask,
		Subtasks:  []Subtask{{Points: 4, StTests: []int{1, 2}}},
		Tests: []Test{
			{Ac: true, Reached: true, Finished: true},
			{Reached: true},
		},
	}

	score := eval.CalculateScore()

	assert.Equal(t, 100, score.ScoreBar.Yellow)
	assert.Equal(t, 0, score.ScoreBar.Red)
}

func TestCalculateScoreMixedGroupsShowPendingUnitAsYellow(t *testing.T) {
	eval := Eval{
		ScoreUnit: ScoreUnitTestGroup,
		Groups: []TestGroup{
			{Points: 4, TgTests: []int{1, 2}},
			{Points: 6, TgTests: []int{3}},
		},
		Tests: []Test{
			{Ac: true, Reached: true, Finished: true},
			{Ac: true, Reached: true, Finished: true},
			{Reached: true},
		},
	}

	score := eval.CalculateScore()

	assert.Equal(t, 40, score.ScoreBar.Green)
	assert.Equal(t, 60, score.ScoreBar.Yellow)
	assert.Equal(t, 0, score.ScoreBar.Red)
}
