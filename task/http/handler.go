package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/patrickmn/go-cache"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/httpjson"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/user/auth"
	"golang.org/x/sync/singleflight"
)

type taskHttpHandler struct {
	taskSrvc srvc.TaskSrvcClient
	cache    *cache.Cache
	sfGroup  singleflight.Group // Added singleflight group to prevent cache stampedes
	exportMu sync.Mutex
}

func NewTaskHttpHandler(taskSrvc srvc.TaskSrvcClient) *taskHttpHandler {
	// Create a cache with 3 second default expiration and 10 second cleanup interval
	c := cache.New(5*time.Second, 10*time.Second)
	return &taskHttpHandler{
		taskSrvc: taskSrvc,
		cache:    c,
		// singleflight.Group doesn't need initialization
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

			r.Group(func(r chi.Router) {
				r.Use(middleware.ThrottleBacklog(1, 2, 30*time.Second))
				r.Get("/tasks/{taskId}/export", h.ExportTask)
				r.Post("/tasks/upload", h.UploadTask)
			})

			// task management
			r.Delete("/tasks/{taskId}", h.DeleteTask)

			// statement
			r.Patch("/tasks/{taskId}/statements/{langIso639}", HttpJsonHandlerNoResp(h.PutStatement))
			r.Post("/tasks/{taskId}/images", h.UploadStatementImage)
			r.Delete("/tasks/{taskId}/images/{filename}", HttpJsonHandlerNoReqNoResp(h.DeleteStatementImage))

			// illustration
			r.Post("/tasks/{taskId}/illustration", h.UploadIllustrationImage)
			r.Delete("/tasks/{taskId}/illustration", h.DeleteIllustrationImage)
		})
	})
}

func (h *taskHttpHandler) logger(ctx context.Context) *slog.Logger {
	return ctxlog.FromContext(ctx).With("module", "task", "layer", "http")
}

type JsonHandlerFuncImpl[Q any, R any] func(ctx context.Context, request Q) (response R, err error)

func HttpJsonHandlerFunc[Q any, R any](handler JsonHandlerFuncImpl[Q, R]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req Q
		t := reflect.TypeOf(req)
		if t.Size() > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				httpjson.BadRequest(w, err.Error())
				return
			}
		}

		result, err := handler(ctx, req)

		if err != nil {
			writeHttpJsonError(w, err)
			return
		}

		httpjson.Success(w, result)
	}
}

// Handler when there is no request body, but there is a JSON response
type JsonHandlerNoReq[R any] func(ctx context.Context) (response R, err error)

func HttpJsonHandlerNoReq[R any](handler JsonHandlerNoReq[R]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result, err := handler(ctx)
		if err != nil {
			writeHttpJsonError(w, err)
			return
		}
		httpjson.Success(w, result)
	}
}

// Handler when there is a request body, but no JSON response body
type JsonHandlerNoResp[Q any] func(ctx context.Context, request Q) (err error)

func HttpJsonHandlerNoResp[Q any](handler JsonHandlerNoResp[Q]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req Q
		t := reflect.TypeOf(req)
		if t.Size() > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				httpjson.BadRequest(w, err.Error())
				return
			}
		}
		if err := handler(ctx, req); err != nil {
			writeHttpJsonError(w, err)
			return
		}
		httpjson.Success(w, struct{}{})
	}
}

// Handler when there is neither request body nor response body
type JsonHandlerNoReqNoResp func(ctx context.Context) (err error)

func HttpJsonHandlerNoReqNoResp(handler JsonHandlerNoReqNoResp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := handler(ctx); err != nil {
			writeHttpJsonError(w, err)
			return
		}
		httpjson.Success(w, struct{}{})
	}
}

func writeHttpJsonError(w http.ResponseWriter, err error) {
	e, ok := err.(*srvcerror.Error)
	if !ok {
		httpjson.InternalError(w)
		return
	}

	msg := e.Error()
	status := httpStatus(*e)
	code := e.ErrorCode()
	httpjson.Error(w, msg, status, code)
}
