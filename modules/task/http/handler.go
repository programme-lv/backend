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

type taskHttpHandler struct {
	taskSrvc srvc.TaskService
	cache    *oldCache.Cache

	getTaskViewCache *cache.LruCache[string, Task]
	getTaskListCache *cache.LruCache[string, []TaskPreview]

	publicAssetStore           *filestore.Store
	testfileStore              *filestore.Store
	testfileDownloadSigningKey []byte
}

type HandlerOption func(*taskHttpHandler)

func WithFileStores(publicAssetStore, testfileStore *filestore.Store, testfileDownloadSigningKey []byte) HandlerOption {
	return func(h *taskHttpHandler) {
		h.publicAssetStore = publicAssetStore
		h.testfileStore = testfileStore
		h.testfileDownloadSigningKey = testfileDownloadSigningKey
	}
}

func NewTaskHttpHandler(taskSrvc srvc.TaskService, opts ...HandlerOption) *taskHttpHandler {
	// Create a cache with 3 second default expiration and 10 second cleanup interval
	c := oldCache.New(5*time.Second, 10*time.Second)
	h := &taskHttpHandler{
		taskSrvc: taskSrvc,
		cache:    c,
		// singleflight.Group doesn't need initialization

		// getTaskViewCache: cache.New(cache.WithJanitorInterval[string, Task](10*time.Second), cache.AsLRU[string, Task](lru.WithCapacity(1000))),
		// getTaskListCache: cache.New(cache.WithJanitorInterval[string, []TaskPreview](10*time.Second), cache.AsLRU[string, []TaskPreview](lru.WithCapacity(1000))),
		getTaskViewCache: cache.NewLruCache[string, Task](1000),
		getTaskListCache: cache.NewLruCache[string, []TaskPreview](1000),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *taskHttpHandler) RegisterRoutes(r *chi.Mux, jwtKey, adminAPIKey []byte, cookieSecure bool) {
	if h.publicAssetStore != nil {
		r.Get("/assets/*", h.ServePublicAsset)
	}
	if h.testfileStore != nil && len(h.testfileDownloadSigningKey) > 0 {
		r.Get("/testfiles/{filename}", h.ServeTestfile)
	}

	r.Group(func(r chi.Router) {
		r.Use(auth.HttpJwtAuthentication(jwtKey, cookieSecure))

		// routes are throttled because of response caching (prevents cache stampede)
		r.Group(func(r chi.Router) {
			r.Use(middleware.ThrottleBacklog(1, 100, 30*time.Second))
			r.Get("/tasks/{taskId}", hf.NoReqJsonResp(h.GetTaskView))
			r.Get("/tasks", hf.NoReqJsonResp(h.GetTaskList))
		})

		// admin-only routes
		r.Group(func(r chi.Router) {
			r.Use(auth.HttpAllowOnlyAdmins(adminAPIKey))

			// resource-intensive routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.ThrottleBacklog(1, 2, 30*time.Second))
				r.Get("/tasks/{taskId}/export", h.ExportTask)
				r.Post("/tasks/upload", h.UploadTask)
			})

			// task management
			r.Delete("/tasks/{taskId}", hf.NoReqNoResp(h.DeleteTask))

			// statement
			r.Patch("/tasks/{taskId}/statements/{langIso639}", hf.JsonReqNoResp(h.PutStatement))
			r.Post("/tasks/{taskId}/images", h.UploadStatementImage)
			r.Delete("/tasks/{taskId}/images/{filename}", hf.NoReqNoResp(h.DeleteStatementImage))

			// illustration
			r.Post("/tasks/{taskId}/illustration", h.UploadIllustration)
			r.Delete("/tasks/{taskId}/illustration", hf.NoReqNoResp(h.DeleteIllustration))
		})
	})
}

func (h *taskHttpHandler) logger(ctx context.Context) *slog.Logger {
	return ctxlog.FromContext(ctx).With("module", "task", "layer", "http")
}
