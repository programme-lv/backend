package http

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/programme-lv/backend/modules/subm/domain"
)

func TestMapSubmEvalExposesExampleTestIDs(t *testing.T) {
	eval := domain.Eval{
		ScoreUnit: domain.ScoreUnitTest,
		Tests: []domain.Test{
			{Ac: true, Reached: true, Finished: true},
			{Example: true, Ac: true, Reached: true, Finished: true},
		},
	}

	mapped := mapSubmEval(eval)

	assert.Equal(t, []int{2}, mapped.ExampleTestIDs)
	assert.Equal(t, "AA", mapped.Verdicts)
	assert.Equal(t, 1, mapped.ScoreInfo.PossibleScore)
	assert.Equal(t, 1, mapped.ScoreInfo.ReceivedScore)
}
