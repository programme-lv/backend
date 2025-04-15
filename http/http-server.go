package http

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/logger"
	http1 "github.com/programme-lv/backend/subm/http"
	taskhttp "github.com/programme-lv/backend/task/http"
	"github.com/programme-lv/backend/user/auth"
	userhttp "github.com/programme-lv/backend/user/http"
)

// HttpReqInfo describes info about HTTP request
type HttpReqInfo struct {
	method    string
	uri       string
	referer   string
	ipaddr    string
	requestID string
	code      int
	written   int64
	duration  time.Duration
	userAgent string
	protocol  string
	tls       bool
}

type HttpIpResolver struct {
	TrustXForwardedFor bool
	TrustXRealIP       bool
}

// resolveIp extracts the client IP address from the request
func (r *HttpIpResolver) resolveIp(req *http.Request) string {
	// Start with RemoteAddr as the most trusted source
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)

	// Check X-Real-IP header if trusted
	if r.TrustXRealIP {
		if realIP := req.Header.Get("X-Real-IP"); realIP != "" {
			ip = realIP
		}
	}

	// Check X-Forwarded-For header if trusted
	if r.TrustXForwardedFor {
		if forwardedFor := req.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			// X-Forwarded-For can contain multiple IPs: client, proxy1, proxy2
			// First address is the original client
			ips := strings.Split(forwardedFor, ",")
			ip = strings.TrimSpace(ips[0])
		}
	}

	return ip
}

// logHTTPReq logs information about the HTTP request
func logHTTPReq(ri *HttpReqInfo) {
	logLevel := slog.LevelInfo
	if ri.code >= 400 {
		logLevel = slog.LevelWarn
	}

	slog.Log(context.Background(), logLevel, "http req info",
		"req-id", ri.requestID,
		"method", ri.method,
		"uri", ri.uri,
		"status", ri.code,
		"written", fmt.Sprintf("%dB", ri.written),
		"duration", ri.duration,
		"ip", ri.ipaddr,
		"referer", ri.referer,
		"user-agent", ri.userAgent,
		"protocol", ri.protocol,
		"tls", ri.tls,
	)
}

// requestLoggerMiddleware returns a middleware that logs HTTP requests
func requestLoggerMiddleware(next http.Handler) http.Handler {
	ipResolver := &HttpIpResolver{
		TrustXForwardedFor: true,
		TrustXRealIP:       true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate request ID
		requestID := uuid.New().String()

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		reqInfo := &HttpReqInfo{
			method:    r.Method,
			uri:       r.URL.String(),
			referer:   r.Header.Get("Referer"),
			userAgent: r.Header.Get("User-Agent"),
			requestID: requestID,
			protocol:  r.Proto,
			tls:       r.TLS != nil,
		}

		reqInfo.ipaddr = ipResolver.resolveIp(r)

		// Create a logger with request ID
		reqLogger := slog.Default().With("req-id", requestID)

		// Add request ID and logger to context
		ctx := context.WithValue(r.Context(), "requestID", requestID)
		ctx = logger.WithLogger(ctx, reqLogger)
		r = r.WithContext(ctx)

		// Capture metrics about the request
		metrics := httpsnoop.CaptureMetrics(next, w, r)

		// Update request info with response data
		reqInfo.code = metrics.Code
		reqInfo.written = metrics.Written
		reqInfo.duration = metrics.Duration

		// Log the request
		logHTTPReq(reqInfo)
	})
}

type HttpServer struct {
	submHttpHandler *http1.SubmHttpHandler
	taskHttpHandler *taskhttp.TaskHttpHandler
	userHttpHandler *userhttp.UserHttpHandler
	execSrvc        *execsrvc.ExecSrvc
	router          *chi.Mux
	JwtKey          []byte
}

func NewHttpServer(
	submHttpHandler *http1.SubmHttpHandler,
	taskHttpHandler *taskhttp.TaskHttpHandler,
	userHttpHandler *userhttp.UserHttpHandler,
	evalSrvc *execsrvc.ExecSrvc,
	jwtKey []byte,
) *HttpServer {
	router := chi.NewRouter()

	// Add request logger middleware
	router.Use(requestLoggerMiddleware)

	// Add stats logger middleware
	statsLogger := newStatsLogger()
	router.Use(statsLogger.middleware)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://programme.lv", "https://www.programme.lv"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           3000,
	}))

	router.Use(auth.GetJwtAuthMiddleware(jwtKey))

	server := &HttpServer{
		submHttpHandler: submHttpHandler,
		userHttpHandler: userHttpHandler,
		taskHttpHandler: taskHttpHandler,
		execSrvc:        evalSrvc,
		router:          router,
		JwtKey:          jwtKey,
	}

	server.routes()

	return server
}

func (httpserver *HttpServer) Start(address string) error {
	return http.ListenAndServe(address, httpserver.router)
}

func (httpserver *HttpServer) routes() {
	r := httpserver.router

	// submission module
	r.Post("/subm", httpserver.submHttpHandler.PostSubm)
	r.Get("/subm", httpserver.submHttpHandler.GetSubmList)
	r.Get("/subm/{subm-uuid}", httpserver.submHttpHandler.GetFullSubm)
	r.Get("/subm/scores/{username}", httpserver.submHttpHandler.GetMaxScorePerTask)

	// user module
	r.Post("/auth/login", httpserver.userHttpHandler.Login)
	r.Post("/users", httpserver.userHttpHandler.Register)

	// task module
	r.Get("/tasks", httpserver.taskHttpHandler.ListTasks)
	r.Get("/tasks/{taskId}", httpserver.taskHttpHandler.GetTask)

	// other
	r.Get("/programming-languages", httpserver.listProgrammingLangs)
	r.Get("/langs", httpserver.listProgrammingLangs)
	r.Get("/subm-updates", httpserver.submHttpHandler.ListenToSubmListUpdates)
	r.Post("/tester/run", httpserver.testerRun)
	r.Get("/tester/run/{evalUuid}", httpserver.testerListen)
	r.Get("/exec/{execUuid}", httpserver.execGet)
}
