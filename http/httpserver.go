package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v2"
	"github.com/golang-jwt/jwt/v5/request"
	"github.com/programme-lv/backend/auth"
	"github.com/programme-lv/backend/subm"
)

type HttpServer struct {
	submSrvc *subm.SubmissionsService
	router   *chi.Mux
	jwtKey   []byte
}

func NewHttpServer(submSrvc *subm.SubmissionsService, jwtKey []byte) *HttpServer {
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

	router.Use(func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, err := request.BearerExtractor{}.ExtractToken(r)
			if err != nil {
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
	})

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://programme.lv", "https://www.programme.lv"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           3000,
	})

	router.Use(corsMiddleware.Handler)

	server := &HttpServer{
		submSrvc: submSrvc,
		router:   router,
	}

	server.routes()

	return server
}

func (httpserver *HttpServer) Start(address string) error {
	return http.ListenAndServe(address, httpserver.router)
}

func (httpserver *HttpServer) routes() {
	httpserver.router.Post("/submissions", httpserver.createSubmission)
	httpserver.router.Get("/submissions", httpserver.listSubmissions)
}
