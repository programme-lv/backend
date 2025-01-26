package submsrvc_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm/submpgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
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
	// test plan:
	// 1. initialize service
	// 2. attempt to get submission, expect error
	// 3. submit solution in c++ for a tests task
	// 4. await submission created event
	// 5. get submission, expect no error
	// 6. cmp submission to expected
	// 7. recreate the service
	// 8. get submission, expect no error
	// 9. cmp submission to expected
	// - submit solution in c++ for a testgroup task
	// - submit solution in python for a testgroup task

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
		enqueue: func(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []execsrvc.TestFile, cpuMs int, memKiB int, checker *string, interactor *string) error {
			t.Logf("enqueue called with execUuid: %s, srcCode: %s, prLangId: %s, tests: %v, cpuMs: %d, memKiB: %d, checker: %v, interactor: %v", execUuid, srcCode, prLangId, tests, cpuMs, memKiB, checker, interactor)
			return nil
		},
	}

	pgPool := newPgDb(t)
	submRepo := submpgrepo.NewPgSubmRepo(pgPool)
	evalRepo := submpgrepo.NewPgEvalRepo(pgPool)
	srvc, err := submsrvc.NewSubmSrvc(userSrvcMock, taskSrvcMock, execSrvcMock, submRepo, evalRepo)
	require.NoError(t, err)
	require.NotNil(t, srvc)

	bg := context.Background()

	submUUID := uuid.New()
	err = srvc.SubmitSol.Handle(bg, submcmd.SubmitSolParams{
		UUID:        submUUID,
		Submission:  "print(sum([int(x) for x in input().split()]))",
		ProgrLangID: "python3.12",
		TaskShortID: "aplusb",
		AuthorUUID:  uuid.New(),
	})
	require.Error(t, err, "expected error because user does not exists")

	ch, err := srvc.SubNewSubm.Handle(bg, submquery.SubNewSubmsParams{})
	require.NoError(t, err)

	submUUID = uuid.New()
	err = srvc.SubmitSol.Handle(bg, submcmd.SubmitSolParams{
		UUID:        submUUID,
		Submission:  "print(sum([int(x) for x in input().split()]))",
		ProgrLangID: "python3.12",
		TaskShortID: "aplusb",
		AuthorUUID:  mockUserUuid,
	})
	require.NoError(t, err)

	submFromCh := <-ch
	require.Equal(t, submFromCh.UUID, submUUID)

	submFromGet, err := srvc.GetSubm.Handle(bg, submquery.GetSubmParams{
		SubmUUID: submUUID,
	})
	require.NoError(t, err)

	require.Less(t, math.Abs(submFromCh.CreatedAt.Sub(submFromGet.CreatedAt).Seconds()), 1.0)
	submFromCh.CreatedAt = submFromGet.CreatedAt
	require.Equal(t, submFromCh, submFromGet)
	require.Equal(t, submFromCh.UUID, submUUID)

	require.NotEqual(t, submFromCh.CurrEvalUUID, uuid.Nil)

	newSubmSrvc, err := submsrvc.NewSubmSrvc(userSrvcMock, taskSrvcMock, execSrvcMock, submRepo, evalRepo)
	require.NoError(t, err)
	require.NotNil(t, newSubmSrvc)

	submFromGet2, err := newSubmSrvc.GetSubm.Handle(bg, submquery.GetSubmParams{
		SubmUUID: submUUID,
	})
	require.NoError(t, err)
	require.Equal(t, submFromGet, submFromGet2)

	evalFromGet, err := newSubmSrvc.GetEval.Handle(bg, submquery.GetEvalParams{
		EvalUUID: submFromGet.CurrEvalUUID,
	})
	require.NoError(t, err)
	require.Equal(t, submFromGet.CurrEvalUUID, evalFromGet.UUID)
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
		cpuMs int,
		memKiB int,
		checker *string,
		interactor *string,
	) error
	listen func(ctx context.Context, evalUuid uuid.UUID) (<-chan execsrvc.Event, error)
}

func (e execSrvcMock) Enqueue(
	ctx context.Context,
	execUuid uuid.UUID,
	srcCode string,
	prLangId string,
	tests []execsrvc.TestFile,
	cpuMs int,
	memKiB int,
	checker *string,
	interactor *string,
) error {
	return e.enqueue(ctx, execUuid, srcCode, prLangId, tests, cpuMs, memKiB, checker, interactor)
}

func (e execSrvcMock) Subscribe(ctx context.Context, evalUuid uuid.UUID) (<-chan execsrvc.Event, error) {
	return e.listen(ctx, evalUuid)
}
