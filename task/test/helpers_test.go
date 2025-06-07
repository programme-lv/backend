package test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/programme-lv/backend/common/testutil"
	"github.com/programme-lv/backend/conf"
	taskhttp "github.com/programme-lv/backend/task/http"
	"github.com/programme-lv/backend/task/repo"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/stretchr/testify/require"
)

func newTaskSrvc(t *testing.T) srvc.TaskSrvcClient {
	t.Helper()
	pool := testutil.MustGetMigratedTestPostgresDb(t)
	repo := repo.NewTaskPgRepo(pool)
	publicS3 := conf.MustGetTestingS3Bucket()
	ts, err := srvc.NewTaskSrvc(repo, publicS3, nil)
	require.NoError(t, err)
	return ts
}

func newTaskHttpHandler(ts srvc.TaskSrvcClient) http.Handler {
	handler := taskhttp.NewTaskHttpHandler(ts)
	router := chi.NewRouter()
	handler.RegisterRoutes(router, []byte("test"))
	return router
}
