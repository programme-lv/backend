package http

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/patrickmn/go-cache"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/user/auth"
	"golang.org/x/sync/singleflight"
)

type taskHttpHandler struct {
	taskSrvc  srvc.TaskSrvcClient
	cache     *cache.Cache
	sfGroup   singleflight.Group // Added singleflight group to prevent cache stampedes
	exportMu  sync.Mutex
	exportDir string
}

func NewTaskHttpHandler(taskSrvc srvc.TaskSrvcClient) *taskHttpHandler {
	// Create a cache with 3 second default expiration and 10 second cleanup interval
	c := cache.New(5*time.Second, 10*time.Second)
	return &taskHttpHandler{
		taskSrvc: taskSrvc,
		cache:    c,
		// singleflight.Group doesn't need initialization
		exportDir: "/home/kp/progr/proglv/tasks/workspace/export",
	}
}

func (h *taskHttpHandler) RegisterRoutes(r *chi.Mux, jwtKey []byte) {
	r.Group(func(r chi.Router) {
		r.Use(auth.HttpJwtAuthentication(jwtKey))
		r.Get("/tasks/{taskId}", h.ViewTask)
		r.Get("/tasks", h.ViewTaskList)

		// admin-only routes
		r.Group(func(r chi.Router) {
			r.Use(auth.HttpJwtAllowOnlyAdmins)

			r.Get("/tasks/{taskId}/export", h.ExportTask)

			// statement
			r.Patch("/tasks/{taskId}/statements/{langIso639}", h.PutStatement)
			r.Post("/tasks/{taskId}/images", h.UploadStatementImage)
			r.Delete("/tasks/{taskId}/images/{filename}", h.DeleteStatementImage)

			// illustration
			r.Post("/tasks/{taskId}/illustration", h.UploadIllustrationImage)
			r.Delete("/tasks/{taskId}/illustration", h.DeleteIllustrationImage)
		})
	})
}

func (h *taskHttpHandler) logger(ctx context.Context) *slog.Logger {
	return ctxlog.FromContext(ctx).With("module", "task", "layer", "http")
}
