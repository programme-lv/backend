package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/httpjson"
)

const taskGetCacheKeyPrefix = "task_get:"

func taskGetCacheKey(taskId string) string {
	return fmt.Sprintf("%s%s", taskGetCacheKeyPrefix, taskId)
}

// ViewTask returns a task by ID
func (httpserver *taskHttpHandler) ViewTask(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	cacheKey := taskGetCacheKey(taskId)

	// Try to get task from cache
	if cachedTask, found := httpserver.cache.Get(cacheKey); found {
		if task, ok := cachedTask.(*Task); ok {
			httpjson.Success(w, task)
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

		response := httpserver.mapTaskResponse(&task)

		// Store in cache for future requests
		httpserver.cache.Set(cacheKey, response, 0) // Use default expiration time

		return response, nil
	})

	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	response, _ := result.(*Task)
	httpjson.Success(w, response)
}

const taskPreviewListCacheKey = "task_preview_list"

type TaskPreview struct {
	ShortId          string             `json:"short_id"`
	FullName         string             `json:"full_name"`
	IllustrImg       *IllustrationImage `json:"illustr_img"`
	DifficultyRating int                `json:"difficulty_rating"`
	OriginOlympiad   string             `json:"origin_olympiad"`
	OriginNote       string             `json:"origin_note"`
	MdStatementStory string             `json:"md_statement_story"`
}

func (h *taskHttpHandler) ViewTaskList(w http.ResponseWriter, r *http.Request) {
	// Try to get from cache
	if cached, found := h.cache.Get(taskPreviewListCacheKey); found {
		if previews, ok := cached.([]TaskPreview); ok {
			httpjson.Success(w, previews)
			return
		}
	}

	// Use singleflight to prevent cache stampede
	result, err, _ := h.sfGroup.Do(taskPreviewListCacheKey, func() (interface{}, error) {
		if cached, found := h.cache.Get(taskPreviewListCacheKey); found {
			if previews, ok := cached.([]TaskPreview); ok {
				return previews, nil
			}
		}
		tasks, err := h.taskSrvc.ListTaskPreviews(r.Context())
		if err != nil {
			return nil, err
		}
		previews := make([]TaskPreview, 0, len(tasks))
		for _, t := range tasks {
			previews = append(previews, h.mapTaskPreview(t))
		}
		h.cache.Set(taskPreviewListCacheKey, previews, 0)
		return previews, nil
	})

	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	previewList, _ := result.([]TaskPreview)
	httpjson.Success(w, previewList)
}
