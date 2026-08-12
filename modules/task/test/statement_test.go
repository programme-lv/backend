//go:build integration

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/programme-lv/backend/modules/user/auth"
	"github.com/stretchr/testify/require"

	taskhttp "github.com/programme-lv/backend/modules/task/http"
)

func GetTask(t *testing.T, h http.Handler, taskId string) *httptest.ResponseRecorder {
	method := http.MethodGet
	url := fmt.Sprintf("/tasks/%s", taskId)

	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func PatchStatement(t *testing.T, h http.Handler, taskId string, req taskhttp.PutStatementReq, token string) *httptest.ResponseRecorder {
	method := http.MethodPatch
	url := fmt.Sprintf("/tasks/%s/statements/lv", taskId)

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	require.NoError(t, err)

	if token != "" {
		httpReq.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httpReq)
	return w
}

func TestPutStatementHttpRequest(t *testing.T) {
	ts := newTaskSrvc(t)
	taskHttpHandler := newTaskHttpHandler(ts)

	createTaskErr := ts.CreateTask(context.Background(), srvc.Task{
		ShortId: "aplusb",
	})
	require.NoError(t, createTaskErr)

	taskBefore, getTaskErr := ts.GetTask(context.Background(), "aplusb")
	require.NoError(t, getTaskErr)
	require.Equal(t, 0, len(taskBefore.MdStatements))

	req := taskhttp.PutStatementReq{
		Story:   "story",
		Input:   "input",
		Output:  "output",
		Notes:   "notes",
		Scoring: "scoring",
		Talk:    "talk",
		Example: "example",
	}

	w := PatchStatement(t, taskHttpHandler, "aplusb", req, "")
	require.Equal(t, http.StatusUnauthorized, w.Code)

	token, err := auth.GenerateJWT(
		"admin",
		"admin@example.com", uuid.Nil,
		[]byte("test"), 24*time.Hour)
	require.NoError(t, err)

	w = PatchStatement(t, taskHttpHandler, "aplusb", req, token)
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
