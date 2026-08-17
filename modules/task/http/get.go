package http

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
)

func (h *taskHttpHandler) GetTaskView(ctx context.Context) (Task, jsonresp.HttpStatusCoder) {
	taskId := chi.URLParamFromCtx(ctx, "taskId")

	if task, ok := h.getTaskViewCache.Get(taskId); ok {
		return task, nil
	}

	t, err := h.taskSrvc.GetTask(ctx, taskId)
	if err != nil {
		return Task{}, err
	}

	response := h.mapTaskResponse(t)
	if response.TaskFullName == "" {
		panic("task full name is empty")
	}

	h.getTaskViewCache.Set(taskId, response, 20*time.Second)
	return response, nil
}

func (h *taskHttpHandler) GetTaskList(ctx context.Context) ([]TaskPreview, jsonresp.HttpStatusCoder) {
	if previews, ok := h.getTaskListCache.Get(""); ok {
		return previews, nil
	}

	tasks, err := h.taskSrvc.ListTaskPreviews(ctx)
	if err != nil {
		return nil, err
	}
	previews := make([]TaskPreview, 0, len(tasks))
	for _, t := range tasks {
		previews = append(previews, h.mapTaskPreview(t))
	}

	h.getTaskListCache.Set("", previews, 20*time.Second)
	return previews, nil
}
