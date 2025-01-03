package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (httpserver *HttpServer) execGet(w http.ResponseWriter, r *http.Request) {
	execUuidStr := chi.URLParam(r, "execUuid")
	execUuid, err := uuid.Parse(execUuidStr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	exec, err := httpserver.evalSrvc.Get(context.TODO(), execUuid)
	if err != nil {
		handleJsonSrvcError(slog.Default(), w, err)
		return
	}

	writeJsonSuccessResponse(w, exec)
}
