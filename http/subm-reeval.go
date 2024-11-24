package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/httplog/v2"
)

func (httpserver *HttpServer) reevaluateSubmission(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type reevaluateSubmissionRequest struct {
		SubmUUID string `json:"subm_uuid"`
	}

	var request reevaluateSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subm, err := httpserver.submSrvc.ReevaluateSubmission(r.Context(), request.SubmUUID)

	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	response := mapBriefSubm(subm)

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
