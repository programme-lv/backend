package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
