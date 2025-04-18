package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/httpjson"
)

func (httpserver *HttpServer) execGet(w http.ResponseWriter, r *http.Request) {
	execUuidStr := chi.URLParam(r, "execUuid")
	execUuid, err := uuid.Parse(execUuidStr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	exec, err := httpserver.execSrvc.Get(context.TODO(), execUuid)
	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	httpjson.WriteSuccessJson(w, exec)
}
