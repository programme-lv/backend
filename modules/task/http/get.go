package http

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

// GetTaskView returns the JSON task for {taskId}.
// The response is cached for 20 seconds.
func (h *taskHttpHandler) GetTaskView(ctx context.Context) (Task, jsonresp.HttpStatusCoder) {
	taskId := chi.URLParamFromCtx(ctx, "taskId")

	if task, ok := h.getTaskViewCache.Get(taskId); ok {
		return h.addViewHighlights(ctx, task)
	}

	t, err := h.taskSrvc.GetTask(ctx, taskId)
	if err != nil {
		return Task{}, err
	}

	response := h.mapTaskResponse(t)
	if response.TaskFullName == "" {
		h.logger(ctx).Error("task full name empty", "task_id", taskId)
		return Task{}, jsonresp.ErrInternalServerError
	}

	h.getTaskViewCache.Set(taskId, response, 20*time.Second)
	return h.addViewHighlights(ctx, response)
}

// GetTaskList returns JSON previews of all tasks.
// Community stats are fetched in one batch on a cache miss.
// The response is cached for 20 seconds.
func (h *taskHttpHandler) GetTaskList(ctx context.Context) ([]TaskPreview, jsonresp.HttpStatusCoder) {
	if previews, ok := h.getTaskListCache.Get(""); ok {
		return h.addHighlights(ctx, previews)
	}

	tasks, err := h.taskSrvc.ListTaskPreviews(ctx)
	if err != nil {
		return nil, err
	}
	previews := make([]TaskPreview, 0, len(tasks))
	for _, t := range tasks {
		previews = append(previews, h.mapTaskPreview(t))
	}

	h.addCommunityStats(ctx, tasks, previews)

	h.getTaskListCache.Set("", previews, 20*time.Second)
	return h.addHighlights(ctx, previews)
}

// addCommunityStats attaches optional stats in one batch call. A stats
// failure is non-fatal: the service has already logged the sanitized cause,
// and the task list is still usable with the field omitted.
func (h *taskHttpHandler) addCommunityStats(
	ctx context.Context,
	tasks []tasksrvc.TaskPreview,
	previews []TaskPreview,
) {
	if h.communityStats == nil || len(tasks) == 0 {
		return
	}

	taskIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ShortId)
	}

	stats, err := h.communityStats.TaskCommunityStats(ctx, taskIDs)
	if err != nil {
		return
	}

	mapped := mapCommunityStats(stats)
	for i := range previews {
		stat, ok := mapped[previews[i].ShortId]
		if !ok {
			continue
		}
		statCopy := stat
		previews[i].CommunityStats = &statCopy
	}
}

// GetTaskFilters returns the origin catalog for the task-list sidebar.
// The response is cached for 20 seconds.
func (h *taskHttpHandler) GetTaskFilters(ctx context.Context) (TaskFilterTree, jsonresp.HttpStatusCoder) {
	if tree, ok := h.getTaskFiltersCache.Get(""); ok {
		return tree, nil
	}

	tree, err := h.taskSrvc.ListTaskFilters(ctx)
	if err != nil {
		return TaskFilterTree{}, err
	}
	response := mapFilterTree(tree)
	h.getTaskFiltersCache.Set("", response, 20*time.Second)
	return response, nil
}
