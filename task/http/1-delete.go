package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/httpjson"
)

// DeleteTask deletes a task and all its related data (admin-only endpoint)
func (h *taskHttpHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := h.logger(ctx)
	taskId := chi.URLParam(r, "taskId")

	if taskId == "" {
		logger.Error("missing task ID parameter")
		httpjson.BadRequest(w, "task ID is required")
		return
	}

	logger.Info("deleting task", "task_id", taskId)

	err := h.taskSrvc.DeleteTask(ctx, taskId)
	if err != nil {
		logger.Error("failed to delete task", "task_id", taskId, "error", err)
		writeSrvcError(w, err)
		return
	}

	// Clear any cached data for this task
	cacheKey := taskGetCacheKey(taskId)
	h.cache.Delete(cacheKey)

	logger.Info("task deleted successfully", "task_id", taskId)
	httpjson.Success(w, map[string]string{
		"message": "task deleted successfully",
		"task_id": taskId,
	})
}
