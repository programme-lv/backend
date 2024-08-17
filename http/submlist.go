package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/programme-lv/backend/subm"
)

func (httpserver *HttpServer) listSubmissions(w http.ResponseWriter, r *http.Request) {
	type listSubmissionsResponse []*submissionResponse
	// TODO: JWT authentification
	subms, err := httpserver.submSrvc.ListSubmissions(context.TODO())
	if err != nil {
		// TODO: better error handling
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
