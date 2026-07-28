package exec_test

// we test integration with tester

import (
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/modules/exec"
	"github.com/stretchr/testify/require"
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}
}

func TestExecResult(t *testing.T) {
	ctx := t.Context()
	repo := exec.NewInMemExecRepo()
	natsConn := conf.MustGetNatsConnFromEnv(ctx)
	srvc := exec.NewExecSrvc(ctx, repo, natsConn)

	pollingErr := srvc.StartPollingResultQueue(ctx)
	require.NoError(t, pollingErr)

	execUuid := uuid.New()
	stdin := "1 2 3"
	answer := "6"
	code := "print(sum(map(int, input().split())))"
	langId := "python3.13"
	tests := []exec.TestFile{{InContent: &stdin, AnsContent: &answer}}
	constraints := exec.TestingParams{CpuMs: 1000, MemKiB: 1024 * 10}

	err := srvc.Enqueue(ctx, execUuid,
		code, langId, tests, constraints)
	require.NoError(t, err)

	res, err := srvc.Get(ctx, execUuid)
	require.NoError(t, err)

	expected := exec.Execution{
		UUID:  execUuid,
		Stage: exec.StageFinished,
		TestRes: []exec.TestRes{
			{
				ID:       1,
				Input:    &stdin,
				Answer:   &answer,
				Reached:  true,
				Ignored:  false,
				Finished: true,
				Subm: &exec.RunData{
					StdIn:       "1 2 3",
					StdOut:      "6\n",
					StdErr:      "",
					CpuMs:       13,
					WallMs:      31,
					MemKiB:      3796,
					ExitCode:    0,
					CtxSwV:      5,
					CtxSwF:      3,
					Signal:      nil,
					IsOomKilled: false,
					IsolStatus:  nil,
					IsolMsg:     nil,
				},
				Checker: &exec.RunData{
					StdIn:       "",
					StdOut:      "",
					StdErr:      "ok \"6\"\n",
					CpuMs:       2,
					WallMs:      11,
					MemKiB:      768,
					ExitCode:    0,
					CtxSwV:      5,
					CtxSwF:      4,
					Signal:      nil,
					IsOomKilled: false,
					IsolStatus:  nil,
					IsolMsg:     nil,
				},
			},
		},
		PrLang: exec.PrLang{
			ShortId:   "python3.13",
			Display:   "Python 3.13",
			CodeFname: "solution.py",
			CompCmd:   nil,
			CompFname: nil,
			ExecCmd:   "python3.13 solution.py",
		},
		Params:    constraints,
		ErrorMsg:  nil,
		SysInfo:   nil,
		SubmComp:  nil,
		CreatedAt: time.Now(),
	}
	similarErr := expected.SimilarTo(res)
	require.NoError(t, similarErr)

	require.Equal(t, expected.UUID, res.UUID)
	require.InDelta(t, expected.CreatedAt.UnixMilli(), res.CreatedAt.UnixMilli(), 2000)
	require.NotEmpty(t, res.SysInfo)

}

func TestEventStream(t *testing.T) {
	ctx := t.Context()
	repo := exec.NewInMemExecRepo()
	natsConn := conf.MustGetNatsConnFromEnv(ctx)
	srvc := exec.NewExecSrvc(ctx, repo, natsConn)

	pollingErr := srvc.StartPollingResultQueue(ctx)
	require.NoError(t, pollingErr)

	execUuid := uuid.New()
	stdin := "1 2 3"
	answer := "6"
	code := "print(sum(map(int, input().split())))"
	langId := "python3.13"
	tests := []exec.TestFile{{InContent: &stdin, AnsContent: &answer}}
	constraints := exec.TestingParams{CpuMs: 1000, MemKiB: 1024 * 10}

	err := srvc.Enqueue(ctx, execUuid,
		code, langId, tests, constraints)
	require.NoError(t, err)

	ch, err := srvc.Listen(ctx, execUuid)
	require.NoError(t, err)

	collected := []exec.Event{}
	for ev := range ch {
		collected = append(collected, ev)
		t.Logf("received event: %s", ev.Type())
	}

	require.Len(t, collected, 4)

	// Event 1: ReceivedSubmission
	receivedEv, ok := collected[0].(exec.ReceivedSubmission)
	require.True(t, ok, "first event = ReceivedSubmission")
	require.NotEmpty(t, receivedEv.SysInfo)
	require.False(t, receivedEv.StartedAt.IsZero())

	// Event 2: ReachedTest
	reachedEv, ok := collected[1].(exec.ReachedTest)
	require.True(t, ok, "second event = ReachedTest")
	require.Equal(t, 1, reachedEv.TestId)
	require.Equal(t, stdin, *reachedEv.In)
	require.Equal(t, answer, *reachedEv.Ans)

	// Event 3: FinishedTest
	finishedTestEv, ok := collected[2].(exec.FinishedTest)
	require.True(t, ok, "third event = FinishedTest")
	require.Equal(t, 1, finishedTestEv.TestID)
	require.NotNil(t, finishedTestEv.Subm)
	require.NotNil(t, finishedTestEv.Checker)

	// Validate submission runtime data
	require.Equal(t, stdin, finishedTestEv.Subm.StdIn)
	require.Equal(t, "6\n", finishedTestEv.Subm.StdOut)
	require.Equal(t, int64(0), finishedTestEv.Subm.ExitCode)
	require.Equal(t, false, finishedTestEv.Subm.IsOomKilled)

	// Validate checker runtime data (testlib checker)
	require.Equal(t, "", finishedTestEv.Checker.StdIn)
	require.Equal(t, "", finishedTestEv.Checker.StdOut)
	require.NotEmpty(t, finishedTestEv.Checker.StdErr) // Should contain "ok"
	require.Equal(t, int64(0), finishedTestEv.Checker.ExitCode)
	require.Equal(t, false, finishedTestEv.Checker.IsOomKilled)

	// Event 4: FinishedTesting
	finishedTestingEv, ok := collected[3].(exec.FinishedTesting)
	require.True(t, ok, "fourth event = FinishedTesting")
	_ = finishedTestingEv
}

func BenchmarkExecResult(b *testing.B) {
	ctx := b.Context()
	repo := exec.NewInMemExecRepo()
	natsConn := conf.MustGetNatsConnFromEnv(ctx)
	srvc := exec.NewExecSrvc(ctx, repo, natsConn)

	pollingErr := srvc.StartPollingResultQueue(ctx)
	require.NoError(b, pollingErr)

	stdin := "1 2 3"
	answer := "6"
	code := "print(sum(map(int, input().split())))"
	langId := "python3.13"
	tests := []exec.TestFile{{InContent: &stdin, AnsContent: &answer}}
	constraints := exec.TestingParams{CpuMs: 1000, MemKiB: 1024 * 10}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		execUuid := uuid.New()
		err := srvc.Enqueue(ctx, execUuid, code, langId, tests, constraints)
		require.NoError(b, err)

		_, err = srvc.Get(ctx, execUuid)
		require.NoError(b, err)
	}
}
