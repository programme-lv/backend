package task_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/task/repo"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/stretchr/testify/require"

	taskhttp "github.com/programme-lv/backend/task/http"
)

func NewTaskSrvc(t *testing.T) srvc.TaskSrvcClient {
	pool := newTestPgDb(t)
	repo := repo.NewTaskPgRepo(pool)
	ts, err := srvc.NewTaskSrvc(repo)
	require.NoError(t, err)
	return ts
}

func NewTaskHttpHandler(t *testing.T, ts srvc.TaskSrvcClient) http.Handler {
	handler := taskhttp.NewTaskHttpHandler(ts)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	return router
}

func GetTask(t *testing.T, h http.Handler, taskId string) *httptest.ResponseRecorder {
	method := http.MethodGet
	url := fmt.Sprintf("/tasks/%s", taskId)

	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func PutStatement(t *testing.T, h http.Handler, taskId string, req taskhttp.PutStatementRequest) *httptest.ResponseRecorder {
	method := http.MethodPut
	url := fmt.Sprintf("/tasks/%s/statements/lv", taskId)

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	require.NoError(t, err)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httpReq)
	return w
}

func TestPutStatementHttpRequest(t *testing.T) {
	ts := NewTaskSrvc(t)
	h := NewTaskHttpHandler(t, ts)

	err := ts.CreateTask(context.Background(), srvc.Task{
		ShortId:        "aplusb",
		FullName:       "a+b",
		OriginNotes:    []srvc.OriginNote{},
		MdStatements:   []srvc.MarkdownStatement{},
		MdImages:       []srvc.StatementImage{},
		PdfStatements:  []srvc.PdfStatement{},
		VisInpSubtasks: []srvc.VisibleInputSubtask{},
		Examples:       []srvc.Example{},
		Tests:          []srvc.Test{},
		Checker:        "",
		Interactor:     "",
		Subtasks:       []srvc.Subtask{},
		TestGroups:     []srvc.TestGroup{},
	})
	require.NoError(t, err)

	taskBefore, err := ts.GetTask(context.Background(), "aplusb")
	require.NoError(t, err)
	require.Equal(t, 0, len(taskBefore.MdStatements))

	req := taskhttp.PutStatementRequest{
		Story:   "story",
		Input:   "input",
		Output:  "output",
		Notes:   "notes",
		Scoring: "scoring",
		Talk:    "talk",
		Example: "example",
	}

	w := PutStatement(t, h, "aplusb", req)
	require.Equal(t, http.StatusOK, w.Code)

	task, err := ts.GetTask(context.Background(), "aplusb")
	require.NoError(t, err)

	require.Equal(t, 1, len(task.MdStatements))

	s := task.MdStatements[0]
	require.Equal(t, "lv", s.LangIso639)
	require.Equal(t, req.Story, s.Story)
	require.Equal(t, req.Input, s.Input)
	require.Equal(t, req.Output, s.Output)
	require.Equal(t, req.Notes, s.Notes)
	require.Equal(t, req.Scoring, s.Scoring)
	require.Equal(t, req.Talk, s.Talk)
	require.Equal(t, req.Example, s.Example)
}
