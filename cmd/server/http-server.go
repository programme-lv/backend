package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/programme-lv/backend/modules/user/auth"
)

type httpRouteRegistrar interface {
	RegisterRoutes(r *chi.Mux)
}

type authenticatedHTTPRouteRegistrar interface {
	RegisterRoutes(r *chi.Mux, jwtKey, adminAPIKey []byte)
}

type httpServer struct {
	submHTTPHandler  authenticatedHTTPRouteRegistrar
	taskHTTPHandler  authenticatedHTTPRouteRegistrar
	userHTTPHandler  httpRouteRegistrar
	execHTTPHandler  httpRouteRegistrar
	plangHTTPHandler httpRouteRegistrar
	router           *chi.Mux
	jwtKey           []byte
	adminAPIKey      []byte
}

func newHTTPServer(
	submHTTPHandler authenticatedHTTPRouteRegistrar,
	taskHTTPHandler authenticatedHTTPRouteRegistrar,
	userHTTPHandler httpRouteRegistrar,
	execHTTPHandler httpRouteRegistrar,
	plangHTTPHandler httpRouteRegistrar,
	jwtKey []byte,
	adminAPIKey []byte,
) *httpServer {
	router := chi.NewRouter()

	router.Use(requestLoggerMiddleware)

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

	router.Use(auth.HttpJwtAuthentication(jwtKey))

	server := &httpServer{
		submHTTPHandler:  submHTTPHandler,
		taskHTTPHandler:  taskHTTPHandler,
		userHTTPHandler:  userHTTPHandler,
		execHTTPHandler:  execHTTPHandler,
		plangHTTPHandler: plangHTTPHandler,
		router:           router,
		jwtKey:           jwtKey,
		adminAPIKey:      adminAPIKey,
	}

	server.routes()

	return server
}

func (s *httpServer) start(address string) error {
	return http.ListenAndServe(address, s.router)
}

func (s *httpServer) routes() {
	s.submHTTPHandler.RegisterRoutes(s.router, s.jwtKey, s.adminAPIKey)
	s.taskHTTPHandler.RegisterRoutes(s.router, s.jwtKey, s.adminAPIKey)
	s.userHTTPHandler.RegisterRoutes(s.router)
	s.execHTTPHandler.RegisterRoutes(s.router)
	s.plangHTTPHandler.RegisterRoutes(s.router)
}
