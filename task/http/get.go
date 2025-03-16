package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/httpjson"
)

const taskGetCacheKeyPrefix = "task_get:"

// GetTask returns a task by ID
func (httpserver *TaskHttpHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	// Apply middleware to the handler
	handler := httpserver.wrapMiddleware(httpserver.getTaskHandler)
	handler(w, r)
}

// getTaskHandler is the actual implementation of GetTask
func (httpserver *TaskHttpHandler) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	cacheKey := fmt.Sprintf("%s%s", taskGetCacheKeyPrefix, taskId)

	// Try to get task from cache
	if cachedTask, found := httpserver.cache.Get(cacheKey); found {
		if task, ok := cachedTask.(*Task); ok {
			httpjson.WriteSuccessJson(w, task)
			return
		}
	}

	// If not in cache or invalid cache, use singleflight to prevent multiple concurrent requests
	// from all hitting the database at the same time
	result, err, _ := httpserver.sfGroup.Do(cacheKey, func() (interface{}, error) {
		// Check cache again in case another request already populated it while we were waiting
		if cachedTask, found := httpserver.cache.Get(cacheKey); found {
			if task, ok := cachedTask.(*Task); ok {
				return task, nil
			}
		}

		// If still not in cache, get from service
		task, err := httpserver.taskSrvc.GetTask(r.Context(), taskId)
		if err != nil {
			return nil, err
		}

		response := mapTaskResponse(&task)

		// Store in cache for future requests
		httpserver.cache.Set(cacheKey, response, 0) // Use default expiration time

		return response, nil
	})

	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	response, _ := result.(*Task)
	httpjson.WriteSuccessJson(w, response)
}
