package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
)

func (httpserver *HttpServer) getSubmission(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	submUuid := chi.URLParam(r, "submUuid")

	subm, err := httpserver.submSrvc.GetSubmission(context.TODO(), submUuid)
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	response := mapFullSubm(subm)

	writeJsonSuccessResponse(w, response)
}
