package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/submsrvc"
)

func (httpserver *HttpServer) listSubmissions(w http.ResponseWriter, r *http.Request) {
	type listSubmissionsResponse []*Submission

	subms, err := httpserver.submSrvc.ListSubms(context.TODO())
	if err != nil {
		handleJsonSrvcError(slog.Default(), w, err)
		return
	}

	mapSubmList := func(subms []submsrvc.Submission) listSubmissionsResponse {
		response := make(listSubmissionsResponse, len(subms))
		for i, subm := range subms {
			response[i] = mapSubm(subm)
		}
		return response
	}

	response := mapSubmList(subms)

	writeJsonSuccessResponse(w, response)
}
