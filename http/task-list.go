package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/httpjson"
)

func (httpserver *HttpServer) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := httpserver.taskSrvc.ListTasks(context.TODO())
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	response := mapTasksResponse(tasks)

	httpjson.WriteSuccessJson(w, response)
}
