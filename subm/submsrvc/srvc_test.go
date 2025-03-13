package submsrvc_test

import (
	"context"
	"math"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/execsrvc"
	submadaptermock "github.com/programme-lv/backend/mocks/submadapter"
	"github.com/programme-lv/backend/subm/submdomain"
	"github.com/programme-lv/backend/subm/submpgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/subm/submsrvc/submadapter"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
	"github.com/programme-lv/backend/task/taskdomain"
	"github.com/programme-lv/backend/usersrvc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var mockUserUuid = uuid.New()

type testSetup struct {
	userSrvc *submadaptermock.MockUserSrvcFacade
	taskSrvc *submadaptermock.MockTaskSrvcFacade
	execSrvc *submadaptermock.MockExecSrvcFacade
	submRepo submadapter.SubmRepo
	evalRepo submadapter.EvalRepo
	srvc     submsrvc.SubmSrvcClient
}

func setupSubmSrvc(t *testing.T) *testSetup {
	t.Helper()
	t.Log("Setting up test dependencies...")

	db := newPgMigratedTestDbConn(t)
	submRepo := submpgrepo.NewPgSubmRepo(db)
	evalRepo := submpgrepo.NewPgEvalRepo(db)

	setup := &testSetup{
		userSrvc: submadaptermock.NewMockUserSrvcFacade(t),
		taskSrvc: submadaptermock.NewMockTaskSrvcFacade(t),
		execSrvc: submadaptermock.NewMockExecSrvcFacade(t),
		submRepo: submRepo,
		evalRepo: evalRepo,
	}

	setup.srvc = submsrvc.NewSubmSrvc(
		setup.userSrvc,
		setup.taskSrvc,
		setup.execSrvc,
		setup.submRepo,
		setup.evalRepo,
	)

	return setup
}

// submitSolution is a helper to submit a solution and return its UUID
func (s *testSetup) submitSolution(t *testing.T, ctx context.Context) uuid.UUID {
	t.Helper()
	s.execSrvc.EXPECT().Enqueue(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.execSrvc.EXPECT().Listen(mock.Anything, mock.Anything).Return(newMockExecEventChannel(t), nil)
	s.userSrvc.EXPECT().GetUserByUUID(mock.Anything, mockUserUuid).Return(usersrvc.User{UUID: mockUserUuid}, nil)
	s.taskSrvc.EXPECT().GetTask(mock.Anything, "aplusb").Return(taskdomain.Task{ShortId: "aplusb"}, nil)
	submUUID := uuid.New()
	err := s.srvc.SubmitSol(ctx, submcmd.SubmitSolParams{
		UUID:        submUUID,
		Submission:  "print(sum([int(x) for x in input().split()]))",
		ProgrLangID: "python3.12",
		TaskShortID: "aplusb",
		AuthorUUID:  mockUserUuid,
	})
	require.NoError(t, err)
	return submUUID
}

// TestSubmitSolutionBasicFlow tests the basic flow of submitting a solution
// and receiving a notification about it.
func TestSubmitSolutionBasicFlow(t *testing.T) {
	setup := setupSubmSrvc(t)
	bg := context.Background()

	t.Log("Setting up submission channel...")
	newSubmCh, err := setup.srvc.SubscribeNewSubms(bg)
	require.NoError(t, err)

	t.Log("Submitting solution...")
	submUUID := setup.submitSolution(t, bg)

	t.Log("Waiting for submission notification...")
	submFromCh := <-newSubmCh
	require.Equal(t, submFromCh.UUID, submUUID)

	t.Log("Fetching submission details...")
	submFromGet, err := setup.srvc.GetSubm(bg, submUUID)
	require.NoError(t, err)

	t.Log("Verifying submission details...")
	require.Less(t, math.Abs(submFromCh.CreatedAt.Sub(submFromGet.CreatedAt).Seconds()), 1.0)
	submFromCh.CreatedAt = submFromGet.CreatedAt
	require.Equal(t, submFromCh, submFromGet)
}

// TestSubmitSolutionPersistence tests that submissions are correctly persisted
// and can be retrieved by a new service instance.
func TestSubmitSolutionPersistence(t *testing.T) {
	setup := setupSubmSrvc(t)
	bg := context.Background()

	t.Log("Submitting solution...")
	submUUID := setup.submitSolution(t, bg)

	t.Log("Fetching original submission...")
	origSubm, err := setup.srvc.GetSubm(bg, submUUID)
	require.NoError(t, err)

	t.Log("Creating new service instance...")
	newSrvc := submsrvc.NewSubmSrvc(
		setup.userSrvc,
		setup.taskSrvc,
		setup.execSrvc,
		setup.submRepo,
		setup.evalRepo,
	)

	t.Log("Verifying submission persistence...")
	persistedSubm, err := newSrvc.GetSubm(bg, submUUID)
	require.NoError(t, err)
	require.Equal(t, origSubm, persistedSubm)

	t.Log("Verifying evaluation persistence...")
	evalFromGet, err := newSrvc.GetEval(bg, origSubm.CurrEvalUUID)
	require.NoError(t, err)
	require.Equal(t, origSubm.CurrEvalUUID, evalFromGet.UUID)
}

// TestSubmitSolutionEvaluation tests the evaluation update flow for a submission.
func TestSubmitSolutionEvaluation(t *testing.T) {
	setup := setupSubmSrvc(t)
	bg := context.Background()

	evalUpdCh, err := setup.srvc.SubscribeEvalUpds(bg)
	require.NoError(t, err)

	t.Log("Submitting solution...")
	submUUID := setup.submitSolution(t, bg)

	t.Log("Fetching submission to get evaluation UUID...")
	subm, err := setup.srvc.GetSubm(bg, submUUID)
	require.NoError(t, err)

	updates := []submdomain.Eval{}
	for update := range evalUpdCh {
		updates = append(updates, update)
		if update.UUID == subm.CurrEvalUUID {
			if update.Stage == submdomain.EvalStageFinished {
				break
			}
		}
	}

	// Verify the sequence of evaluation stages
	expectedStages := []string{
		"waiting",
		"compiling",
		"compiling",
		"testing",
		"testing",
		"testing",
		"testing",
		"testing",
		"finished",
	}

	for i, update := range updates {
		require.Equal(t, expectedStages[i], string(update.Stage), "Unexpected evaluation stage at index %d", i)
	}
}

// TestUserMaxScores tests that GetMaxScorePerTask correctly calculates maximum scores
func TestUserMaxScores(t *testing.T) {
	setup := setupSubmSrvc(t)
	bg := context.Background()

	// Initially there should be no scores
	scores, err := setup.srvc.GetMaxScorePerTask(bg, mockUserUuid)
	require.NoError(t, err)
	require.Equal(t, scores, map[string]submdomain.MaxScore{})

	// Submit a solution that will get 50% score (1 out of 2 tests pass)
	setup.execSrvc.EXPECT().Enqueue(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockCh := make(chan execsrvc.Event, 10)
	setup.execSrvc.EXPECT().Listen(mock.Anything, mock.Anything).Return(mockCh, nil).Once()
	setup.userSrvc.EXPECT().GetUserByUUID(mock.Anything, mockUserUuid).Return(usersrvc.User{UUID: mockUserUuid}, nil).Once()
	setup.taskSrvc.EXPECT().GetTask(mock.Anything, "aplusb").Return(newAplusBTask(t), nil).Once()
	setup.taskSrvc.EXPECT().GetTestDownlUrl(mock.Anything, mock.Anything).Return("", nil)

	submUUID := uuid.New()
	err = setup.srvc.SubmitSol(bg, submcmd.SubmitSolParams{
		UUID:        submUUID,
		Submission:  "print(sum([int(x) for x in input().split()]))",
		ProgrLangID: "python3.12",
		TaskShortID: "aplusb",
		AuthorUUID:  mockUserUuid,
	})
	require.NoError(t, err)

	subm, err := setup.srvc.GetSubm(bg, submUUID)
	require.NoError(t, err)

	// Simulate evaluation events
	mockCh <- execsrvc.ReceivedSubmission{SysInfo: "test", StartedAt: time.Now()}
	mockCh <- execsrvc.StartedTesting{}
	mockCh <- execsrvc.ReachedTest{TestId: 1}
	mockCh <- execsrvc.FinishedTest{TestID: 1, Subm: &execsrvc.RunData{ExitCode: 0}, Checker: &execsrvc.RunData{ExitCode: 0}} // AC
	mockCh <- execsrvc.ReachedTest{TestId: 2}
	mockCh <- execsrvc.FinishedTest{TestID: 2, Subm: &execsrvc.RunData{ExitCode: 0}, Checker: &execsrvc.RunData{ExitCode: 1}} // WA
	mockCh <- execsrvc.FinishedTesting{}
	close(mockCh)

	err = setup.srvc.WaitForEvalFinish(bg, subm.CurrEvalUUID)
	require.NoError(t, err)

	subms, err := setup.srvc.ListSubms(bg, submquery.ListSubmsParams{
		Limit:  10000,
		Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, subms, 1)
	require.Equal(t, subms[0].UUID, subm.UUID)

	eval, err := setup.srvc.GetEval(bg, subm.CurrEvalUUID)
	require.NoError(t, err)
	require.Equal(t, eval.Stage, submdomain.EvalStageFinished)

	score := eval.CalculateScore()
	require.Equal(t, score.ReceivedScore, 1)
	require.Equal(t, score.PossibleScore, 2)

	// Check scores - should have 50% on aplusb
	scores, err = setup.srvc.GetMaxScorePerTask(bg, mockUserUuid)
	require.NoError(t, err)
	require.Equal(t, map[string]submdomain.MaxScore{
		"aplusb": {
			SubmUuid: subm.UUID,
			Received: 1,
			Possible: 2,
		},
	}, scores)

	// Submit another solution that gets 100% score
	mockCh2 := make(chan execsrvc.Event, 10)
	setup.execSrvc.EXPECT().Enqueue(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	setup.execSrvc.EXPECT().Listen(mock.Anything, mock.Anything).Return(mockCh2, nil).Once()
	setup.userSrvc.EXPECT().GetUserByUUID(mock.Anything, mockUserUuid).Return(usersrvc.User{UUID: mockUserUuid}, nil)
	setup.taskSrvc.EXPECT().GetTask(mock.Anything, "aplusb").Return(newAplusBTask(t), nil)
	setup.taskSrvc.EXPECT().GetTestDownlUrl(mock.Anything, mock.Anything).Return("", nil)

	submUUID2 := uuid.New()
	err = setup.srvc.SubmitSol(bg, submcmd.SubmitSolParams{
		UUID:        submUUID2,
		Submission:  "print(sum([int(x) for x in input().split()]))", // Same code but this time both tests pass
		ProgrLangID: "python3.12",
		TaskShortID: "aplusb",
		AuthorUUID:  mockUserUuid,
	})
	require.NoError(t, err)

	subm2, err := setup.srvc.GetSubm(bg, submUUID2)
	require.NoError(t, err)

	// Simulate evaluation events - both tests pass this time
	mockCh2 <- execsrvc.ReceivedSubmission{SysInfo: "test", StartedAt: time.Now()}
	mockCh2 <- execsrvc.StartedTesting{}
	mockCh2 <- execsrvc.ReachedTest{TestId: 1}
	mockCh2 <- execsrvc.FinishedTest{TestID: 1, Subm: &execsrvc.RunData{ExitCode: 0}, Checker: &execsrvc.RunData{ExitCode: 0}} // AC
	mockCh2 <- execsrvc.ReachedTest{TestId: 2}
	mockCh2 <- execsrvc.FinishedTest{TestID: 2, Subm: &execsrvc.RunData{ExitCode: 0}, Checker: &execsrvc.RunData{ExitCode: 0}} // AC
	mockCh2 <- execsrvc.FinishedTesting{}
	close(mockCh2)

	err = setup.srvc.WaitForEvalFinish(bg, subm2.CurrEvalUUID)
	require.NoError(t, err)

	eval2, err := setup.srvc.GetEval(bg, subm2.CurrEvalUUID)
	require.NoError(t, err)
	require.Equal(t, eval2.Stage, submdomain.EvalStageFinished)

	score2 := eval2.CalculateScore()
	require.Equal(t, score2.ReceivedScore, 2)
	require.Equal(t, score2.PossibleScore, 2)

	// Check scores - should now have 100% on aplusb
	scores, err = setup.srvc.GetMaxScorePerTask(bg, mockUserUuid)
	require.NoError(t, err)
	require.Equal(t, map[string]submdomain.MaxScore{
		"aplusb": {
			SubmUuid: subm2.UUID, // Should be the second submission since it has a higher score
			Received: 2,
			Possible: 2,
		},
	}, scores)
}

func newAplusBTask(t *testing.T) taskdomain.Task {
	t.Helper()
	return taskdomain.Task{
		ShortId:         "aplusb",
		MemLimMegabytes: 256,
		CpuTimeLimSecs:  1,
		Tests: []taskdomain.Test{
			{
				InpSha2: "a8692502350d26305a557cf6277fe8594130c73b7aaeb24ed5413335dd6daa8c",
				AnsSha2: "030e27d5723736abfbdd64046cfeacf2d9f6f52c3fb1638a0cdcbe95d1ab87c2",
			},
			{
				InpSha2: "07bf895436232279171deb4fda0fe2b11e3df5e8d309d4a0be1841ea4f942e61",
				AnsSha2: "f44e60ad9601f0c79ec56031c81f07cdd27cf3dab473005d3c1abbd451140036",
			},
		},
	}
}

// newMockExecEventChannel creates and returns a channel that simulates the execution service events
func newMockExecEventChannel(t *testing.T) <-chan execsrvc.Event {
	t.Helper()
	t.Log("Creating mock execution event channel...")
	mockCh := make(chan execsrvc.Event, 1)
	go func() {
		defer close(mockCh)
		events := []execsrvc.Event{
			execsrvc.ReceivedSubmission{
				SysInfo:   "some sys info",
				StartedAt: time.Now(),
			},
			execsrvc.StartedCompiling{},
			execsrvc.FinishedCompiling{
				RuntimeData: getExampleRunData(),
			},
			execsrvc.StartedTesting{},
			execsrvc.ReachedTest{
				TestId: 1,
				In:     getExampleStrPtr(),
				Ans:    getExampleStrPtr(),
			},
			execsrvc.FinishedTest{
				TestID:  1,
				Subm:    getExampleRunData(),
				Checker: getExampleRunData(),
			},
			execsrvc.ReachedTest{
				TestId: 2,
				In:     getExampleStrPtr(),
				Ans:    getExampleStrPtr(),
			},
			execsrvc.FinishedTest{
				TestID:  2,
				Subm:    getExampleRunData(),
				Checker: getExampleRunData(),
			},
			execsrvc.FinishedTesting{},
		}
		for _, event := range events {
			t.Logf("Sending mock event: %T", event)
			mockCh <- event
		}
	}()
	return mockCh
}

// Helper that generates random run data for tests
func getExampleRunData() *execsrvc.RunData {
	return &execsrvc.RunData{
		StdIn:    "some std in",
		StdOut:   "some std out",
		StdErr:   "some std err",
		CpuMs:    1 + int64(rand.IntN(100)),
		WallMs:   2 + int64(rand.IntN(100)),
		MemKiB:   3 + int64(rand.IntN(100)),
		ExitCode: 4 + int64(rand.IntN(100)),
		CtxSwV:   rand.Int64(),
		CtxSwF:   rand.Int64(),
		Signal:   nil,
	}
}

// Helper that generates random string pointer
func getExampleStrPtr() *string {
	s := strconv.Itoa(rand.IntN(100))
	return &s
}

// newPgMigratedTestDbConn returns a connection pool to a unique and isolated test database fully migrated and ready for testing
func newPgMigratedTestDbConn(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "proglv", // local dev pg user
		Password:   "proglv", // local dev pg password
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	gm := golangmigrator.New("../../migrate")
	config := pgtestdb.Custom(t, conf, gm)

	pool, err := pgxpool.New(ctx, config.URL())
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
	})

	pool.Exec(
		ctx,
		"INSERT INTO users (uuid, firstname, lastname, username, email, bcrypt_pwd, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		mockUserUuid,
		"John", "Doe",
		"johndoe",
		"johndoe@example.com",
		"password",
		time.Now(),
	)

	return pool
}
