package http

import (
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/httpjson"
)

const taskListCacheKey = "task_list"

// ListTasks returns a list of all tasks
func (httpserver *TaskHttpHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Try to get tasks from cache
	if cachedTasks, found := httpserver.cache.Get(taskListCacheKey); found {
		if tasks, ok := cachedTasks.([]*Task); ok {
			httpjson.WriteSuccessJson(w, tasks)
			return
		}
	}

	// If not in cache or invalid cache, use singleflight to prevent multiple concurrent requests
	// from all hitting the database at the same time
	result, err, _ := httpserver.sfGroup.Do(taskListCacheKey, func() (interface{}, error) {
		// Check cache again in case another request already populated it while we were waiting
		if cachedTasks, found := httpserver.cache.Get(taskListCacheKey); found {
			if tasks, ok := cachedTasks.([]*Task); ok {
				return tasks, nil
			}
		}

		// If still not in cache, get from service
		tasks, err := httpserver.taskSrvc.ListTasks(r.Context())
		if err != nil {
			return nil, err
		}

		response := mapTasksResponse(tasks)

		// Store in cache for future requests
		httpserver.cache.Set(taskListCacheKey, response, 0) // Use default expiration time

		return response, nil
	})

	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	response, _ := result.([]*Task)
	httpjson.WriteSuccessJson(w, response)
}
