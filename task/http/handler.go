package http

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/patrickmn/go-cache"
	"github.com/programme-lv/backend/task/srvc"
	"golang.org/x/sync/singleflight"
)

type TaskHttpHandler struct {
	taskSrvc srvc.TaskSrvcClient
	cache    *cache.Cache
	sfGroup  singleflight.Group // Added singleflight group to prevent cache stampedes
}

func NewTaskHttpHandler(taskSrvc srvc.TaskSrvcClient) *TaskHttpHandler {
	// Create a cache with 3 second default expiration and 10 second cleanup interval
	c := cache.New(5*time.Second, 10*time.Second)
	return &TaskHttpHandler{
		taskSrvc: taskSrvc,
		cache:    c,
		// singleflight.Group doesn't need initialization
	}
}

func (h *TaskHttpHandler) RegisterRoutes(r *chi.Mux) {
	r.Get("/tasks", h.ListTasks)
	r.Get("/tasks/{taskId}", h.GetTask)
	r.Put("/tasks/{taskId}/statements/{langIso639}", h.UpdateStatement)
}
