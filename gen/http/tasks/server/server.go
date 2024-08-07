// Code generated by goa v3.18.2, DO NOT EDIT.
//
// tasks HTTP server
//
// Command:
// $ goa gen github.com/programme-lv/backend/design

package server

import (
	"context"
	"net/http"

	tasks "github.com/programme-lv/backend/gen/tasks"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
	"goa.design/plugins/v3/cors"
)

// Server lists the tasks service endpoint HTTP handlers.
type Server struct {
	Mounts    []*MountPoint
	ListTasks http.Handler
	GetTask   http.Handler
	CORS      http.Handler
}

// MountPoint holds information about the mounted endpoints.
type MountPoint struct {
	// Method is the name of the service method served by the mounted HTTP handler.
	Method string
	// Verb is the HTTP method used to match requests to the mounted handler.
	Verb string
	// Pattern is the HTTP request path pattern used to match requests to the
	// mounted handler.
	Pattern string
}

// New instantiates HTTP handlers for all the tasks service endpoints using the
// provided encoder and decoder. The handlers are mounted on the given mux
// using the HTTP verb and path defined in the design. errhandler is called
// whenever a response fails to be encoded. formatter is used to format errors
// returned by the service methods prior to encoding. Both errhandler and
// formatter are optional and can be nil.
func New(
	e *tasks.Endpoints,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) goahttp.Statuser,
) *Server {
	return &Server{
		Mounts: []*MountPoint{
			{"ListTasks", "GET", "/tasks"},
			{"GetTask", "GET", "/tasks/{task_id}"},
			{"CORS", "OPTIONS", "/tasks"},
			{"CORS", "OPTIONS", "/tasks/{task_id}"},
		},
		ListTasks: NewListTasksHandler(e.ListTasks, mux, decoder, encoder, errhandler, formatter),
		GetTask:   NewGetTaskHandler(e.GetTask, mux, decoder, encoder, errhandler, formatter),
		CORS:      NewCORSHandler(),
	}
}

// Service returns the name of the service served.
func (s *Server) Service() string { return "tasks" }

// Use wraps the server handlers with the given middleware.
func (s *Server) Use(m func(http.Handler) http.Handler) {
	s.ListTasks = m(s.ListTasks)
	s.GetTask = m(s.GetTask)
	s.CORS = m(s.CORS)
}

// MethodNames returns the methods served.
func (s *Server) MethodNames() []string { return tasks.MethodNames[:] }

// Mount configures the mux to serve the tasks endpoints.
func Mount(mux goahttp.Muxer, h *Server) {
	MountListTasksHandler(mux, h.ListTasks)
	MountGetTaskHandler(mux, h.GetTask)
	MountCORSHandler(mux, h.CORS)
}

// Mount configures the mux to serve the tasks endpoints.
func (s *Server) Mount(mux goahttp.Muxer) {
	Mount(mux, s)
}

// MountListTasksHandler configures the mux to serve the "tasks" service
// "listTasks" endpoint.
func MountListTasksHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleTasksOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/tasks", f)
}

// NewListTasksHandler creates a HTTP handler which loads the HTTP request and
// calls the "tasks" service "listTasks" endpoint.
func NewListTasksHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) goahttp.Statuser,
) http.Handler {
	var (
		encodeResponse = EncodeListTasksResponse(encoder)
		encodeError    = goahttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "listTasks")
		ctx = context.WithValue(ctx, goa.ServiceKey, "tasks")
		var err error
		res, err := endpoint(ctx, nil)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountGetTaskHandler configures the mux to serve the "tasks" service
// "getTask" endpoint.
func MountGetTaskHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleTasksOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/tasks/{task_id}", f)
}

// NewGetTaskHandler creates a HTTP handler which loads the HTTP request and
// calls the "tasks" service "getTask" endpoint.
func NewGetTaskHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeGetTaskRequest(mux, decoder)
		encodeResponse = EncodeGetTaskResponse(encoder)
		encodeError    = goahttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "getTask")
		ctx = context.WithValue(ctx, goa.ServiceKey, "tasks")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountCORSHandler configures the mux to serve the CORS endpoints for the
// service tasks.
func MountCORSHandler(mux goahttp.Muxer, h http.Handler) {
	h = HandleTasksOrigin(h)
	mux.Handle("OPTIONS", "/tasks", h.ServeHTTP)
	mux.Handle("OPTIONS", "/tasks/{task_id}", h.ServeHTTP)
}

// NewCORSHandler creates a HTTP handler which returns a simple 204 response.
func NewCORSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
}

// HandleTasksOrigin applies the CORS response headers corresponding to the
// origin for the service tasks.
func HandleTasksOrigin(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Not a CORS request
			h.ServeHTTP(w, r)
			return
		}
		if cors.MatchOrigin(origin, "http://localhost:3000") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Expose-Headers", "*")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if acrm := r.Header.Get("Access-Control-Request-Method"); acrm != "" {
				// We are handling a preflight request
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(204)
				return
			}
			h.ServeHTTP(w, r)
			return
		}
		if cors.MatchOrigin(origin, "https://programme.lv") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Expose-Headers", "*")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if acrm := r.Header.Get("Access-Control-Request-Method"); acrm != "" {
				// We are handling a preflight request
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(204)
				return
			}
			h.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
		return
	})
}
