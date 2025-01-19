package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm/submhttp"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"
)

type HttpServer struct {
	submHttpServer *submhttp.SubmHttpServer
	submSrvc       *submsrvc.SubmSrvc
	userSrvc       *usersrvc.UserSrvc
	taskSrvc       *tasksrvc.TaskSrvc
	evalSrvc       *execsrvc.ExecSrvc
	router         *chi.Mux
	JwtKey         []byte
}

type endpointStats struct {
	count       int
	totalTime   time.Duration
	lastPrinted time.Time
}

type statsLogger struct {
	stats         map[string]*endpointStats
	mu            sync.Mutex
	flushInterval time.Duration
}

func newStatsLogger() *statsLogger {
	sl := &statsLogger{
		stats:         make(map[string]*endpointStats),
		flushInterval: 5 * time.Second, // Print stats every 5 seconds
	}
	go sl.periodicFlush()
	return sl
}

func (sl *statsLogger) periodicFlush() {
	ticker := time.NewTicker(sl.flushInterval)
	for range ticker.C {
		sl.flushStats()
	}
}

func (sl *statsLogger) flushStats() {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	now := time.Now()
	for endpoint, stats := range sl.stats {
		if stats.count > 0 && now.Sub(stats.lastPrinted) >= sl.flushInterval {
			// Convert to float64 and round to 2 decimal places
			avgTimeMs := float64(stats.totalTime.Microseconds()) / float64(stats.count) / 1000.0

			slog.Info("endpoint stats",
				"endpoint", endpoint,
				"count", stats.count,
				"avg_time_ms", fmt.Sprintf("%.2f", avgTimeMs),
				"period", sl.flushInterval,
			)
			// Reset stats after printing
			stats.count = 0
			stats.totalTime = 0
			stats.lastPrinted = now
		}
	}
}

func (sl *statsLogger) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)

		sl.mu.Lock()
		if _, exists := sl.stats[endpoint]; !exists {
			sl.stats[endpoint] = &endpointStats{}
		}
		sl.stats[endpoint].count++
		sl.stats[endpoint].totalTime += duration
		sl.mu.Unlock()
	})
}

func NewHttpServer(
	submHttpServer *submhttp.SubmHttpServer,
	submSrvc *submsrvc.SubmSrvc,
	userSrvc *usersrvc.UserSrvc,
	taskSrvc *tasksrvc.TaskSrvc,
	evalSrvc *execsrvc.ExecSrvc,
	jwtKey []byte,
) *HttpServer {
	router := chi.NewRouter()

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

	router.Use(getJwtAuthMiddleware(jwtKey))

	server := &HttpServer{
		submHttpServer: submHttpServer,
		submSrvc:       submSrvc,
		userSrvc:       userSrvc,
		taskSrvc:       taskSrvc,
		evalSrvc:       evalSrvc,
		router:         router,
		JwtKey:         jwtKey,
	}

	server.routes()

	return server
}

func (httpserver *HttpServer) Start(address string) error {
	return http.ListenAndServe(address, httpserver.router)
}

func (httpserver *HttpServer) routes() {
	r := httpserver.router
	// r.Post("/submissions", httpserver.createSubmission)
	// r.Post("/reevaluate", httpserver.reevaluateSubmissions)
	r.Get("/subm", httpserver.submHttpServer.GetSubmList)
	r.Get("/subm/{subm-uuid}", httpserver.submHttpServer.GetSubmView)
	r.Post("/auth/login", httpserver.authLogin)
	r.Post("/users", httpserver.authRegister)
	r.Get("/tasks", httpserver.listTasks)
	r.Get("/tasks/{taskId}", httpserver.getTask)
	r.Get("/programming-languages", httpserver.listProgrammingLangs)
	r.Get("/langs", httpserver.listProgrammingLangs)
	// r.Get("/subm-updates", httpserver.listenToSubmListUpdates)
	r.Post("/tester/run", httpserver.testerRun)
	r.Get("/tester/run/{evalUuid}", httpserver.testerListen)
	r.Get("/exec/{execUuid}", httpserver.execGet)
}
