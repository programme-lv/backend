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
	srvc, err := submsrvc.NewSubmSrvc(taskSrvc, evalSrvc)
	require.NoError(t, err)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	submCh, err := srvc.ListenToNewSubmCreated(ctx)
	require.NoError(t, err)
	subm, err := srvc.CreateSubmission(ctx, &submsrvc.CreateSubmissionParams{
		Submission: "a,b=input().split();print(int(a)+int(b))",
		Username:   "admin",
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
		eval = &evalUpd.Eval
		stage = eval.Stage
	}
	require.Equal(t, submsrvc.StageFinished, stage)
	require.Equal(t, submsrvc.ScoreUnitTest, eval.ScoreUnit)
	require.Equal(t, submsrvc.Test{
		Ac:       true,
		Wa:       false,
		Tle:      false,
		Mle:      false,
		Re:       false,
		Ig:       false,
		Reached:  true,
		Finished: true,
	}, eval.Tests[0])
	require.Equal(t, submsrvc.Test{
		Ac:       true,
		Wa:       false,
		Tle:      false,
		Mle:      false,
		Re:       false,
		Ig:       false,
		Reached:  true,
		Finished: true,
	}, eval.Tests[1])
	require.Nil(t, eval.Error)
	require.NotNil(t, eval)
	subm.CurrEval = eval
	submSrvc2, err := submsrvc.NewSubmSrvc(taskSrvc, evalSrvc)
	require.NoError(t, err)
	submFromGet, err := submSrvc2.GetSubm(ctx, subm.UUID)
	require.NoError(t, err)
	require.Equal(t, subm, submFromGet)
}
