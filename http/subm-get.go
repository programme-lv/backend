package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
)

func (httpserver *HttpServer) getSubmission(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "submUuid")

	logger := httplog.LogEntry(r.Context())

	subm, err := httpserver.submSrvc.GetSubmission(context.TODO(), taskId)
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	response := mapFullSubm(subm)

	writeJsonSuccessResponse(w, response)
}
