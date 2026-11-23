package http

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/programme-lv/backend/common/srvcerror"
	submsrvc "github.com/programme-lv/backend/modules/subm/srvc"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTaskPreviewService struct {
	tasksrvc.TaskService
	previews []tasksrvc.TaskPreview
	calls    int
}

func (f *fakeTaskPreviewService) ListTaskPreviews(context.Context) ([]tasksrvc.TaskPreview, srvcerror.E) {
	f.calls++
	return f.previews, nil
}

type fakeCommunityStatsService struct {
	stats map[string]submsrvc.TaskCommunityStats
	err   srvcerror.E
	calls int
	ids   []string
}

func (f *fakeCommunityStatsService) TaskCommunityStats(
	_ context.Context,
	taskShortIDs []string,
) (map[string]submsrvc.TaskCommunityStats, srvcerror.E) {
	f.calls++
	f.ids = append([]string(nil), taskShortIDs...)
	return f.stats, f.err
}

func taskListWithTwoPreviews() []tasksrvc.TaskPreview {
	return []tasksrvc.TaskPreview{
		{ShortId: "task-a", FullName: map[string]string{"lv": "A"}},
		{ShortId: "task-b", FullName: map[string]string{"lv": "B"}},
	}
}

func TestGetTaskListMapsCommunityStatsAndOmitsEmptyTasks(t *testing.T) {
	taskSrvc := &fakeTaskPreviewService{previews: taskListWithTwoPreviews()}
	statsSrvc := &fakeCommunityStatsService{
		stats: map[string]submsrvc.TaskCommunityStats{
			"task-a": {
				UniqueSolvers:   1,
				UniqueAttempted: 3,
				Verdicts: map[string]int{
					submsrvc.CommunityVerdictAccepted:         1,
					submsrvc.CommunityVerdictWrongAnswer:      1,
					submsrvc.CommunityVerdictPartiallyCorrect: 1,
					// Zero counts are not part of the public payload.
					submsrvc.CommunityVerdictCompileError: 0,
				},
			},
		},
	}
	h := NewTaskHttpHandler(taskSrvc, WithCommunityStats(statsSrvc))

	previews, err := h.GetTaskList(context.Background())
	require.NoError(t, err)
	require.Len(t, previews, 2)
	assert.Equal(t, []string{"task-a", "task-b"}, statsSrvc.ids)

	require.NotNil(t, previews[0].CommunityStats)
	assert.Equal(t, 1, previews[0].CommunityStats.UniqueSolvers)
	assert.Equal(t, 3, previews[0].CommunityStats.UniqueAttempted)
	assert.Equal(t, map[string]int{
		submsrvc.CommunityVerdictAccepted:         1,
		submsrvc.CommunityVerdictWrongAnswer:      1,
		submsrvc.CommunityVerdictPartiallyCorrect: 1,
	}, previews[0].CommunityStats.Verdicts)
	assert.Nil(t, previews[1].CommunityStats)

	raw, marshalErr := json.Marshal(previews)
	require.NoError(t, marshalErr)
	var decoded []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Len(t, decoded, 2)
	assert.Contains(t, decoded[0], "community_stats")
	assert.NotContains(t, decoded[1], "community_stats")
}

func TestGetTaskListOmitsCommunityStatsOnReaderError(t *testing.T) {
	taskSrvc := &fakeTaskPreviewService{previews: taskListWithTwoPreviews()}
	statsSrvc := &fakeCommunityStatsService{err: srvcerror.InternalServerError()}
	h := NewTaskHttpHandler(taskSrvc, WithCommunityStats(statsSrvc))

	previews, err := h.GetTaskList(context.Background())
	require.NoError(t, err)
	require.Len(t, previews, 2)
	assert.Nil(t, previews[0].CommunityStats)
	assert.Nil(t, previews[1].CommunityStats)

	raw, marshalErr := json.Marshal(previews)
	require.NoError(t, marshalErr)
	assert.NotContains(t, string(raw), "community_stats")
}

func TestGetTaskListWithoutCommunityStats(t *testing.T) {
	taskSrvc := &fakeTaskPreviewService{previews: taskListWithTwoPreviews()}
	h := NewTaskHttpHandler(taskSrvc)

	previews, err := h.GetTaskList(context.Background())
	require.NoError(t, err)
	require.Len(t, previews, 2)
	assert.Nil(t, previews[0].CommunityStats)
	assert.Nil(t, previews[1].CommunityStats)
}

func TestGetTaskListCachesCommunityStats(t *testing.T) {
	taskSrvc := &fakeTaskPreviewService{previews: taskListWithTwoPreviews()}
	statsSrvc := &fakeCommunityStatsService{
		stats: map[string]submsrvc.TaskCommunityStats{
			"task-a": {UniqueAttempted: 1, Verdicts: map[string]int{"wrong_answer": 1}},
		},
	}
	h := NewTaskHttpHandler(taskSrvc, WithCommunityStats(statsSrvc))

	first, err := h.GetTaskList(context.Background())
	require.NoError(t, err)
	second, err := h.GetTaskList(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, taskSrvc.calls)
	assert.Equal(t, 1, statsSrvc.calls)
	require.Len(t, first, len(second))
	assert.Equal(t, first[0].CommunityStats, second[0].CommunityStats)
}

func TestMapCommunityStatsDropsZeroAttempts(t *testing.T) {
	mapped := mapCommunityStats(map[string]submsrvc.TaskCommunityStats{
		"empty":  {UniqueAttempted: 0, Verdicts: map[string]int{"unknown": 0}},
		"task-a": {UniqueSolvers: 1, UniqueAttempted: 1, Verdicts: map[string]int{"accepted": 1}},
	})
	require.Contains(t, mapped, "task-a")
	assert.NotContains(t, mapped, "empty")
	assert.Equal(t, map[string]int{"accepted": 1}, mapped["task-a"].Verdicts)
	assert.Nil(t, mapCommunityStats(nil))
}
