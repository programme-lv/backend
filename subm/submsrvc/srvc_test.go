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
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submpgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"
	"github.com/stretchr/testify/require"
)

var mockUserUuid = uuid.New()

// newPgDb returns a connection pool to a unique and isolated test database fully migrated and ready for testing
func newPgDb(t *testing.T) *pgxpool.Pool {
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

func TestSubmitSolution(t *testing.T) {
	userSrvcMock := userSrvcMock{
		getUserByUUID: func(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error) {
			t.Logf("getUserByUUID called with uuid: %s", uuid)
			return usersrvc.User{UUID: mockUserUuid}, nil
		},
	}

	taskSrvcMock := taskSrvcMock{
		getTask: func(ctx context.Context, shortId string) (tasksrvc.Task, error) {
			t.Logf("getTask called with shortId: %s", shortId)
			return tasksrvc.Task{}, nil
		},
	}

	execSrvcMock := execSrvcMock{
		enqueue: func(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []execsrvc.TestFile, params execsrvc.TestingParams) error {
			cpuMs := params.CpuMs
			memKiB := params.MemKiB
			checker := params.Checker
			interactor := params.Interactor
			t.Logf("enqueue called with execUuid: %s, srcCode: %s, prLangId: %s, tests: %v, cpuMs: %d, memKiB: %d, checker: %v, interactor: %v", execUuid, srcCode, prLangId, tests, cpuMs, memKiB, checker, interactor)
			return nil
		},
		listen: func(ctx context.Context, evalUuid uuid.UUID) (<-chan execsrvc.Event, error) {
			mockCh := make(chan execsrvc.Event, 1000)
			go func() {
				defer close(mockCh)
				events := []execsrvc.Event{
					execsrvc.ReceivedSubmission{ // 1st update to user
						SysInfo:   "some sys info",
						StartedAt: time.Now(),
					},
					execsrvc.StartedCompiling{}, // 2nd update to user
					execsrvc.FinishedCompiling{
						RuntimeData: getExampleRunData(),
					},
					execsrvc.StartedTesting{}, // 3rd update to user
					execsrvc.ReachedTest{ // 4th update to user
						TestId: 1,
						In:     getExampleStrPtr(),
						Ans:    getExampleStrPtr(),
					},
					execsrvc.FinishedTest{ // 5th update to user
						TestID:  1,
						Subm:    getExampleRunData(),
						Checker: getExampleRunData(),
					},
					execsrvc.ReachedTest{ // 6th update to user
						TestId: 2,
						In:     getExampleStrPtr(),
						Ans:    getExampleStrPtr(),
					},
					execsrvc.FinishedTest{ // 7th update to user
						TestID:  2,
						Subm:    getExampleRunData(),
						Checker: getExampleRunData(),
					},
					execsrvc.FinishedTesting{}, // 8th update to user
				}
				for _, event := range events {
					mockCh <- event
				}
			}()
			return mockCh, nil
		},
	}

	pgPool := newPgDb(t)
	submRepo := submpgrepo.NewPgSubmRepo(pgPool)
	evalRepo := submpgrepo.NewPgEvalRepo(pgPool)
	srvc := submsrvc.NewSubmSrvc(userSrvcMock, taskSrvcMock, execSrvcMock, submRepo, evalRepo)
	require.NotNil(t, srvc)

	bg := context.Background()

	submUUID := uuid.New()
	err := srvc.SubmitSol(bg, submcmd.SubmitSolParams{
		UUID:        submUUID,
		Submission:  "print(sum([int(x) for x in input().split()]))",
		ProgrLangID: "python3.12",
		TaskShortID: "aplusb",
		AuthorUUID:  uuid.New(),
	})
	require.NoError(t, err)

	newSubmCh, err := srvc.SubsNewSubm(bg)
	require.NoError(t, err)

	// okay so the next step is to listen for evaluation execution events
	// we have to do that on the original service, because it has the events in mem
	evalUpdCh, err := srvc.SubsEvalUpd(bg)
	require.NoError(t, err)

	submUUID = uuid.New()
	err = srvc.SubmitSol(bg, submcmd.SubmitSolParams{
		UUID:        submUUID,
		Submission:  "print(sum([int(x) for x in input().split()]))",
		ProgrLangID: "python3.12",
		TaskShortID: "aplusb",
		AuthorUUID:  mockUserUuid,
	})
	require.NoError(t, err)

	submFromCh := <-newSubmCh
	require.Equal(t, submFromCh.UUID, submUUID)

	submFromGet, err := srvc.GetSubm(bg, submUUID)
	require.NoError(t, err)

	require.Less(t, math.Abs(submFromCh.CreatedAt.Sub(submFromGet.CreatedAt).Seconds()), 1.0)
	submFromCh.CreatedAt = submFromGet.CreatedAt
	require.Equal(t, submFromCh, submFromGet)
	require.Equal(t, submFromCh.UUID, submUUID)

	require.NotEqual(t, submFromCh.CurrEvalUUID, uuid.Nil)

	newSubmSrvc := submsrvc.NewSubmSrvc(userSrvcMock, taskSrvcMock, execSrvcMock, submRepo, evalRepo)
	require.NotNil(t, newSubmSrvc)

	submFromGet2, err := newSubmSrvc.GetSubm(bg, submUUID)
	require.NoError(t, err)
	require.Equal(t, submFromGet, submFromGet2)

	evalFromGet, err := newSubmSrvc.GetEval(bg, submFromGet.CurrEvalUUID)
	require.NoError(t, err)
	require.Equal(t, submFromGet.CurrEvalUUID, evalFromGet.UUID)
	evalUpdates := []subm.Eval{}
	timeout := time.After(5 * time.Second)
	for {
		select {
		case e, ok := <-evalUpdCh:
			if !ok {
				goto done
			}
			evalUpdates = append(evalUpdates, e)
		case <-timeout:
			t.Fatal("timed out waiting for eval updates")
		}
	}
done:
	require.Equal(t, 8, len(evalUpdates))

}

type userSrvcMock struct {
	getUserByUUID func(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error)
}

func (u userSrvcMock) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error) {
	return u.getUserByUUID(ctx, uuid)
}

type taskSrvcMock struct {
	getTask func(ctx context.Context, shortId string) (tasksrvc.Task, error)
}

func (t taskSrvcMock) GetTask(ctx context.Context, shortId string) (tasksrvc.Task, error) {
	return t.getTask(ctx, shortId)
}

type execSrvcMock struct {
	enqueue func(
		ctx context.Context,
		execUuid uuid.UUID,
		srcCode string,
		prLangId string,
		tests []execsrvc.TestFile,
		params execsrvc.TestingParams,
	) error
	listen func(ctx context.Context, evalUuid uuid.UUID) (<-chan execsrvc.Event, error)
}

func (e execSrvcMock) Enqueue(
	ctx context.Context,
	execUuid uuid.UUID,
	srcCode string,
	prLangId string,
	tests []execsrvc.TestFile,
	params execsrvc.TestingParams,
) error {
	return e.enqueue(ctx, execUuid, srcCode, prLangId, tests, params)
}

func (e execSrvcMock) Listen(ctx context.Context, evalUuid uuid.UUID) (<-chan execsrvc.Event, error) {
	return e.listen(ctx, evalUuid)
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
