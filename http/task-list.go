package http

import (
	"context"
	"log/slog"
	"net/http"
)

func (httpserver *HttpServer) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := httpserver.taskSrvc.ListTasks(context.TODO())
	if err != nil {
		handleJsonSrvcError(slog.Default(), w, err)
		return
	}

	response := mapTasksResponse(tasks)

	writeJsonSuccessResponse(w, response)
}
