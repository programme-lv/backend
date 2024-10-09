package http

import (
	"context"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/submsrvc"
)

func (httpserver *HttpServer) listSubmissions(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type listSubmissionsResponse []*BriefSubmission

	subms, err := httpserver.submSrvc.ListSubmissions(context.TODO())
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	mapSubmList := func(subms []*submsrvc.Submission) listSubmissionsResponse {
		response := make(listSubmissionsResponse, len(subms))
		for i, subm := range subms {
			response[i] = mapBriefSubm(subm)
		}
		return response
	}

	response := mapSubmList(subms)

	writeJsonSuccessResponse(w, response)
}
