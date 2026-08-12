//go:build integration

package test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/testutil"
	taskhttp "github.com/programme-lv/backend/modules/task/http"
	"github.com/programme-lv/backend/modules/task/repo"
	"github.com/programme-lv/backend/modules/task/srvc"
)

func newTaskSrvc(t *testing.T) srvc.TaskService {
	t.Helper()
	pool := testutil.MustGetMigratedTestPostgresDb(t)
	repo := repo.NewTaskPgRepo(pool)
	publicStore, err := filestore.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	testfileStore, err := filestore.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ts := srvc.NewTaskSrvc(repo, publicStore, testfileStore)
	return ts
}

func newTaskHttpHandler(ts srvc.TaskService) http.Handler {
	handler := taskhttp.NewTaskHttpHandler(ts)
	router := chi.NewRouter()
	handler.RegisterRoutes(router, []byte("test"), []byte("test-admin-api-key"), false, nil)
	return router
}
