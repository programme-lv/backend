package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/subm"
)

type HttpServer struct {
	submSrvc *subm.SubmissionsService
	router   *chi.Mux
}

func NewHttpServer(submSrvc *subm.SubmissionsService) *HttpServer {
	router := chi.NewRouter()
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
	httpserver.router.Post("/createSubmission", httpserver.createSubmission)
}
