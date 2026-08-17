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
	RegisterRoutes(r *chi.Mux, jwtKey, adminAPIKey []byte, cookieSecure bool, pwdChangedAt auth.PasswordChangedAtLookup)
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
	cookieSecure     bool
	pwdChangedAt     auth.PasswordChangedAtLookup
}

func newHTTPServer(
	submHTTPHandler authenticatedHTTPRouteRegistrar,
	taskHTTPHandler authenticatedHTTPRouteRegistrar,
	userHTTPHandler httpRouteRegistrar,
	execHTTPHandler httpRouteRegistrar,
	plangHTTPHandler httpRouteRegistrar,
	jwtKey []byte,
	adminAPIKey []byte,
	cookieSecure bool,
	pwdChangedAt auth.PasswordChangedAtLookup,
) *httpServer {
	router := chi.NewRouter()

	router.Use(requestLoggerMiddleware)
	router.Use(corsHandler())

	router.Use(auth.HttpJwtAuthentication(
		jwtKey,
		auth.WithSecureCookie(cookieSecure),
		auth.WithPasswordChangedAtLookup(pwdChangedAt),
	))

	server := &httpServer{
		submHTTPHandler:  submHTTPHandler,
		taskHTTPHandler:  taskHTTPHandler,
		userHTTPHandler:  userHTTPHandler,
		execHTTPHandler:  execHTTPHandler,
		plangHTTPHandler: plangHTTPHandler,
		router:           router,
		jwtKey:           jwtKey,
		adminAPIKey:      adminAPIKey,
		cookieSecure:     cookieSecure,
		pwdChangedAt:     pwdChangedAt,
	}

	server.routes()

	return server
}

func corsHandler() func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://programme.lv", "https://www.programme.lv"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           3000,
	})
}

func (s *httpServer) start(address string) error {
	return http.ListenAndServe(address, s.router)
}

func (s *httpServer) routes() {
	s.router.Handle("/metrics", metricsHandler())

	s.submHTTPHandler.RegisterRoutes(s.router, s.jwtKey, s.adminAPIKey, s.cookieSecure, s.pwdChangedAt)
	s.taskHTTPHandler.RegisterRoutes(s.router, s.jwtKey, s.adminAPIKey, s.cookieSecure, s.pwdChangedAt)
	s.userHTTPHandler.RegisterRoutes(s.router)
	s.execHTTPHandler.RegisterRoutes(s.router)
	s.plangHTTPHandler.RegisterRoutes(s.router)
}
