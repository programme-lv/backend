package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/httpjson"
)

const taskGetCacheKeyPrefix = "task_get:"

func (httpserver *TaskHttpHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	cacheKey := fmt.Sprintf("%s%s", taskGetCacheKeyPrefix, taskId)

	// Try to get task from cache
	if cachedTask, found := httpserver.cache.Get(cacheKey); found {
		if task, ok := cachedTask.(*Task); ok {
			httpjson.WriteSuccessJson(w, task)
			return
		}
	}

	// If not in cache or invalid cache, get from service
	task, err := httpserver.taskSrvc.GetTask(context.TODO(), taskId)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	response := mapTaskResponse(&task)

	// Store in cache for future requests
	httpserver.cache.Set(cacheKey, response, 0) // Use default expiration time

	httpjson.WriteSuccessJson(w, response)
}
