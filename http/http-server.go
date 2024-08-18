package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v2"
	"github.com/golang-jwt/jwt/v5/request"
	"github.com/programme-lv/backend/auth"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/user"
)

type HttpServer struct {
	submSrvc *subm.SubmissionSrvc
	userSrvc *user.UserService
	router   *chi.Mux
}

func NewHttpServer(
	submSrvc *subm.SubmissionSrvc,
	userSrvc *user.UserService,
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
		router:   router,
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
	r.Post("/auth/login", httpserver.authLogin)
	r.Post("/users", httpserver.authRegister)
}

func getJwtAuthMiddleware(jwtKey []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, err := request.BearerExtractor{}.ExtractToken(r)
			if err != nil {
				if errors.Is(err, request.ErrNoTokenInRequest) {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := auth.ValidateJWT(token, jwtKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			fmt.Printf("claims: %+v\n", claims)

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(hfn)
	}
}
