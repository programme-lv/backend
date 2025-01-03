package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (httpserver *HttpServer) getSubmission(w http.ResponseWriter, r *http.Request) {
	submUuidStr := chi.URLParam(r, "submUuid")
	submUuid, err := uuid.Parse(submUuidStr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subm, err := httpserver.submSrvc.GetSubm(context.TODO(), submUuid)
	if err != nil {
		handleJsonSrvcError(slog.Default(), w, err)
		return
	}

	response := mapSubm(*subm)

	writeJsonSuccessResponse(w, response)
}
