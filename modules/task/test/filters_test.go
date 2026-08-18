//go:build integration

package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/programme-lv/backend/common/jsonresp"
	taskhttp "github.com/programme-lv/backend/modules/task/http"
	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/stretchr/testify/require"
)

func TestGetTaskFilters(t *testing.T) {
	ts := newTaskSrvc(t)
	h := newTaskHttpHandler(ts)
	ctx := context.Background()

	require.NoError(t, ts.CreateTask(ctx, srvc.Task{
		ShortId:         "liojun",
		FullName:        map[string]string{"lv": "LIO junior"},
		OriginOlympiad:  "LIO",
		OriginYear:      "2024/2025",
		OlympStage:      "national",
		OriginDivisions: []string{"junior"},
	}))
	require.NoError(t, ts.CreateTask(ctx, srvc.Task{
		ShortId:        "plain",
		FullName:       map[string]string{"lv": "Cits"},
		OriginOlympiad: "",
	}))

	req := httptest.NewRequest(http.MethodGet, "/task-filters", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp jsonresp.JsonResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "success", resp.Status)

	raw, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var tree taskhttp.TaskFilterTree
	require.NoError(t, json.Unmarshal(raw, &tree))
	require.Len(t, tree.Olympiads, 2)
	require.Equal(t, "LIO", tree.Olympiads[0].ID)
	require.Equal(t, 1, tree.Olympiads[0].Count)
	require.Equal(t, "2025", tree.Olympiads[0].Years[0].ID)
	require.Equal(t, "national", tree.Olympiads[0].Years[0].Stages[0].ID)
	require.Equal(t, "junior", tree.Olympiads[0].Years[0].Stages[0].Divisions[0].ID)
	require.Equal(t, "other", tree.Olympiads[1].ID)
	require.Equal(t, 1, tree.Olympiads[1].Count)

	listReq := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)
	var listResp jsonresp.JsonResponse
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listResp))
	listRaw, err := json.Marshal(listResp.Data)
	require.NoError(t, err)
	var previews []taskhttp.TaskPreview
	require.NoError(t, json.Unmarshal(listRaw, &previews))
	byID := map[string]taskhttp.TaskPreview{}
	for _, preview := range previews {
		byID[preview.ShortId] = preview
	}
	require.Equal(t, "LIO", byID["liojun"].OriginOlympiad)
	require.Equal(t, "2025", byID["liojun"].OriginYear)
	require.Equal(t, "national", byID["liojun"].OlympStage)
	require.Equal(t, []string{"junior"}, byID["liojun"].OriginDivisions)
	require.Equal(t, "other", byID["plain"].OriginOlympiad)

	swallowed := httptest.NewRequest(http.MethodGet, "/tasks/filters", nil)
	swallowedW := httptest.NewRecorder()
	h.ServeHTTP(swallowedW, swallowed)
	require.Equal(t, http.StatusNotFound, swallowedW.Code)
}
