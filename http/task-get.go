package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
)

func (httpserver *HttpServer) getTask(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")

	task, err := httpserver.taskSrvc.GetTask(context.TODO(), taskId)
	if err != nil {
		handleJsonSrvcError(httplog.LogEntry(r.Context()), w, err)
		return
	}

	response := mapTaskResponse(&task)

	writeJsonSuccessResponse(w, response)
}
