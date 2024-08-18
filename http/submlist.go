package http

import (
	"context"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/subm"
)

func (httpserver *HttpServer) listSubmissions(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type listSubmissionsResponse []*submissionResponse

	subms, err := httpserver.submSrvc.ListSubmissions(context.TODO())
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	mapSubmissionsResponse := func(subms []*subm.Submission) listSubmissionsResponse {
		response := make(listSubmissionsResponse, len(subms))
		for i, subm := range subms {
			response[i] = mapSubmissionResponse(subm)
		}
		return response
	}

	response := mapSubmissionsResponse(subms)

	writeJsonSuccessResponse(w, response)
}
