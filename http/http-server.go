package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/submsrvc"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/user"
)

type HttpServer struct {
	submSrvc *submsrvc.SubmissionSrvc
	userSrvc *user.UserService
	taskSrvc *tasksrvc.TaskService
	router   *chi.Mux
	JwtKey   []byte
}

func NewHttpServer(
	submSrvc *submsrvc.SubmissionSrvc,
	userSrvc *user.UserService,
	taskSrvc *tasksrvc.TaskService,
	jwtKey []byte,
) *HttpServer {
	router := chi.NewRouter()

	logger := httplog.NewLogger("proglv", httplog.Options{
		LogLevel:         slog.LevelDebug,
		Concise:          true,
		RequestHeaders:   true,
		MessageFieldName: "message",
		Tags: map[string]string{
			"version": "v1.0-81aa4244d9fc8076a",
			"env":     "dev",
		},
	})

	router.Use(httplog.RequestLogger(logger))

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
		submSrvc: submSrvc,
		userSrvc: userSrvc,
		taskSrvc: taskSrvc,
		router:   router,
		JwtKey:   jwtKey,
	}

	server.routes()

	return server
}

func (httpserver *HttpServer) Start(address string) error {
	return http.ListenAndServe(address, httpserver.router)
}

func (httpserver *HttpServer) routes() {
	r := httpserver.router
	r.Post("/submissions", httpserver.createSubmission)
	r.Get("/submissions", httpserver.listSubmissions)
	r.Get("/submissions/{submUuid}", httpserver.getSubmission)
	r.Post("/auth/login", httpserver.authLogin)
	r.Post("/users", httpserver.authRegister)
	r.Get("/tasks", httpserver.listTasks)
	r.Get("/tasks/{taskId}", httpserver.getTask)
	r.Get("/programming-languages", httpserver.listProgrammingLangs)
	r.Get("/subm-updates", httpserver.listenToSubmUpdates)
}
