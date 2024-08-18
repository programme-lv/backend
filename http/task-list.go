package http

import (
	"context"
	"net/http"

	"github.com/go-chi/httplog/v2"
)

func (httpserver *HttpServer) listTasks(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	tasks, err := httpserver.taskSrvc.ListTasks(context.TODO())
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	response := mapTasksResponse(tasks)

	writeJsonSuccessResponse(w, response)
}
