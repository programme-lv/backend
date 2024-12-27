package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
	"github.com/google/uuid"
)

func (httpserver *HttpServer) getSubmission(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	submUuidStr := chi.URLParam(r, "submUuid")
	submUuid, err := uuid.Parse(submUuidStr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subm, err := httpserver.submSrvc.GetSubm(context.TODO(), submUuid)
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	response := mapSubm(*subm)

	writeJsonSuccessResponse(w, response)
}
