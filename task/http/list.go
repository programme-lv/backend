package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/httpjson"
)

const taskListCacheKey = "task_list"

func (httpserver *TaskHttpHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Try to get tasks from cache
	if cachedTasks, found := httpserver.cache.Get(taskListCacheKey); found {
		if tasks, ok := cachedTasks.([]*Task); ok {
			httpjson.WriteSuccessJson(w, tasks)
			return
		}
	}

	// If not in cache or invalid cache, get from service
	tasks, err := httpserver.taskSrvc.ListTasks(context.TODO())
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	response := mapTasksResponse(tasks)

	// Store in cache for future requests
	httpserver.cache.Set(taskListCacheKey, response, 0) // Use default expiration time

	httpjson.WriteSuccessJson(w, response)
}
