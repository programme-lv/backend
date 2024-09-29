package http

import (
	"context"
	"net/http"

	"github.com/go-chi/httplog/v2"
)

func (httpserver *HttpServer) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := httpserver.taskSrvc.ListTasks(context.TODO())
	if err != nil {
		logger := httplog.LogEntry(r.Context())
		handleJsonSrvcError(logger, w, err)
		return
	}

	response := mapTasksResponse(tasks)

	writeJsonSuccessResponse(w, response)
}
