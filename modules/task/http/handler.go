// Package http is the HTTP gateway for the task module.
//
// It marshals JSON, enforces JWT and admin auth, caches GET responses,
// and maps service errors to HTTP status codes.
// Construct a handler with [NewTaskHttpHandler] and mount it with RegisterRoutes.
package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	oldCache "github.com/patrickmn/go-cache"
	"github.com/programme-lv/backend/common/cache"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/filestore"
	hf "github.com/programme-lv/backend/common/httpfunc"
	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/programme-lv/backend/modules/user/auth"
)

// taskHttpHandler serves the task HTTP API.
type taskHttpHandler struct {
	taskSrvc srvc.TaskService
	cache    *oldCache.Cache

	getTaskViewCache *cache.LruCache[string, Task]
	getTaskListCache *cache.LruCache[string, []TaskPreview]

	publicAssetStore           *filestore.Store
	testfileStore              *filestore.Store
	testfileDownloadSigningKey []byte
}

// A HandlerOption configures a task HTTP handler.
type HandlerOption func(*taskHttpHandler)

// WithFileStores sets the public asset store, the test-file store,
// and the HMAC key used to sign test-file download URLs.
func WithFileStores(publicAssetStore, testfileStore *filestore.Store, testfileDownloadSigningKey []byte) HandlerOption {
	return func(h *taskHttpHandler) {
		h.publicAssetStore = publicAssetStore
		h.testfileStore = testfileStore
		h.testfileDownloadSigningKey = testfileDownloadSigningKey
	}
}

// NewTaskHttpHandler returns a task HTTP handler that uses taskSrvc.
func NewTaskHttpHandler(taskSrvc srvc.TaskService, opts ...HandlerOption) *taskHttpHandler {
	c := oldCache.New(5*time.Second, 10*time.Second)
	h := &taskHttpHandler{
		taskSrvc:         taskSrvc,
		cache:            c,
		getTaskViewCache: cache.NewLruCache[string, Task](1000),
		getTaskListCache: cache.NewLruCache[string, []TaskPreview](1000),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RegisterRoutes mounts task HTTP routes on r.
// GET /tasks and GET /tasks/{taskId} require a JWT and are throttled
// to one in-flight request to avoid a cache stampede.
// Admin routes require an admin API key.
// Upload and export are throttled separately because they are expensive.
func (h *taskHttpHandler) RegisterRoutes(r *chi.Mux, jwtKey, adminAPIKey []byte, cookieSecure bool, pwdChangedAt auth.PasswordChangedAtLookup) {
	if h.publicAssetStore != nil {
		r.Get("/assets/*", h.ServePublicAsset)
	}
	if h.testfileStore != nil && len(h.testfileDownloadSigningKey) > 0 {
		r.Get("/testfiles/{filename}", h.ServeTestfile)
	}

	r.Group(func(r chi.Router) {
		r.Use(auth.HttpJwtAuthentication(
			jwtKey,
			auth.WithSecureCookie(cookieSecure),
			auth.WithPasswordChangedAtLookup(pwdChangedAt),
		))

		r.Group(func(r chi.Router) {
			r.Use(middleware.ThrottleBacklog(1, 100, 30*time.Second))
			r.Get("/tasks/{taskId}", hf.NoReqJsonResp(h.GetTaskView))
			r.Get("/tasks", hf.NoReqJsonResp(h.GetTaskList))
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.HttpAllowOnlyAdmins(adminAPIKey))

			r.Group(func(r chi.Router) {
				r.Use(middleware.ThrottleBacklog(1, 2, 30*time.Second))
				r.Get("/tasks/{taskId}/export", h.ExportTask)
				r.Post("/tasks/upload", h.UploadTask)
			})

			r.Delete("/tasks/{taskId}", hf.NoReqNoResp(h.DeleteTask))

			r.Patch("/tasks/{taskId}/statements/{langIso639}", hf.JsonReqNoResp(h.PutStatement))
			r.Post("/tasks/{taskId}/images", h.UploadStatementImage)
			r.Delete("/tasks/{taskId}/images/{filename}", hf.NoReqNoResp(h.DeleteStatementImage))

			r.Post("/tasks/{taskId}/illustration", h.UploadIllustration)
			r.Delete("/tasks/{taskId}/illustration", hf.NoReqNoResp(h.DeleteIllustration))
		})
	})
}

func (h *taskHttpHandler) logger(ctx context.Context) *slog.Logger {
	return ctxlog.FromContext(ctx).With("module", "task", "layer", "http")
}
