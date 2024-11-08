package http

import (
	"net/http"
)

type TesterRunResponse struct {
	EvalUUID string `json:"eval_uuid"`
}

func (httpserver *HttpServer) testerRun(w http.ResponseWriter, r *http.Request) {
	// task, err := httpserver.taskSrvc.GetTask(context.TODO(), taskId)
	// if err != nil {
	// 	handleJsonSrvcError(httplog.LogEntry(r.Context()), w, err)
	// 	return
	// }

	// this should be handled by the submission service
	// or maybe we could handle it right here

	response := &TesterRunResponse{
		EvalUUID: "123",
	}

	writeJsonSuccessResponse(w, response)
}
