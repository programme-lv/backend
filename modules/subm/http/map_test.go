package http

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/programme-lv/backend/modules/subm/domain"
)

func TestEvalStatus(t *testing.T) {
	tests := []struct {
		name string
		eval domain.Eval
		want string
	}{
		{
			name: "stage without error",
			eval: domain.Eval{Stage: domain.EvalStageTesting},
			want: "testing",
		},
		{
			name: "compilation error",
			eval: domain.Eval{
				Stage: domain.EvalStageFinished,
				Error: &domain.EvalError{Type: domain.ErrorTypeCompilation},
			},
			want: "compile_error",
		},
		{
			name: "tester internal error",
			eval: domain.Eval{
				Stage: domain.EvalStageFinished,
				Error: &domain.EvalError{Type: domain.ErrorTypeInternal},
			},
			want: "internal_error",
		},
		{
			name: "tester unavailable",
			eval: domain.Eval{
				Stage: domain.EvalStageFinished,
				Error: &domain.EvalError{Type: domain.ErrorTypeTesterUnavailable},
			},
			want: "tester_unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, evalStatus(tt.eval))
		})
	}
}

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
