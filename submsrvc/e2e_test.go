package submsrvc_test

import (
	"context"
	"testing"
	"time"

	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/submsrvc"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/stretchr/testify/require"
)

func TestSubmSrvc(t *testing.T) {
	taskSrvc, err := tasksrvc.NewTaskSrvc()
	require.NoError(t, err)
	evalSrvc := evalsrvc.NewEvalSrvc()
	srvc := submsrvc.NewSubmSrvc(taskSrvc, evalSrvc)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	submCh, err := srvc.ListenToNewSubmCreated(ctx)
	require.NoError(t, err)
	subm, err := srvc.CreateSubmission(ctx, &submsrvc.CreateSubmissionParams{
		Submission: "a=int(input());b=int(input());print(a+b)",
		Username:   "KrisjanisP",
		ProgLangID: "python3.10",
		TaskCodeID: "aplusb",
	})
	require.NoError(t, err)
	require.NotNil(t, subm)
	submFromCh := <-submCh
	require.Equal(t, subm, submFromCh)
	// retrieve updates for submission until evaluation is marked as finished
	evalCh, err := srvc.ListenToLatestSubmEvalUpdate(ctx, subm.UUID)
	require.NoError(t, err)
	stage := subm.CurrEval.Stage
	var eval *submsrvc.Evaluation
	for stage != submsrvc.StageFinished {
		evalUpd, ok := <-evalCh
		require.True(t, ok)
		require.Equal(t, subm.UUID, evalUpd.SubmUuid)
		eval = evalUpd.Eval
		stage = eval.Stage
	}
	require.Equal(t, submsrvc.StageFinished, stage)
	require.Equal(t, submsrvc.ScoreUnitSubtask, eval.ScoreUnit)
	require.Equal(t, submsrvc.ScoreUnitSubtask, eval.ScoreUnit)
}
