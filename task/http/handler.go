package http

import (
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/programme-lv/backend/task/srvc"
	"golang.org/x/sync/singleflight"
)

type TaskHttpHandler struct {
	taskSrvc   srvc.TaskSrvcClient
	cache      *cache.Cache
	middleware []func(http.Handler) http.Handler
	sfGroup    singleflight.Group // Added singleflight group to prevent cache stampedes
}

func NewTaskHttpHandler(taskSrvc srvc.TaskSrvcClient) *TaskHttpHandler {
	// Create a cache with 3 second default expiration and 10 second cleanup interval
	c := cache.New(5*time.Second, 10*time.Second)
	return &TaskHttpHandler{
		taskSrvc:   taskSrvc,
		cache:      c,
		middleware: []func(http.Handler) http.Handler{},
		// singleflight.Group doesn't need initialization
	}
}

// UseMiddleware adds a middleware to the handler
func (h *TaskHttpHandler) UseMiddleware(middleware func(http.Handler) http.Handler) {
	h.middleware = append(h.middleware, middleware)
}

// wrapMiddleware wraps a handler with all registered middleware
func (h *TaskHttpHandler) wrapMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	// Convert the handler func to an http.Handler
	var next http.Handler = handler

	// Apply middleware in reverse order (last added, first executed)
	for i := len(h.middleware) - 1; i >= 0; i-- {
		next = h.middleware[i](next)
	}

	// Convert back to http.HandlerFunc
	return func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}
}
