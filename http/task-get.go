package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (httpserver *HttpServer) getTask(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")

	task, err := httpserver.taskSrvc.GetTask(context.TODO(), taskId)
	if err != nil {
		handleJsonSrvcError(slog.Default(), w, err)
		return
	}

	response := mapTaskResponse(&task)

	writeJsonSuccessResponse(w, response)
}
