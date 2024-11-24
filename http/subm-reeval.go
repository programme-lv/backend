package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/submsrvc"
)

func (httpserver *HttpServer) reevaluateSubmission(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type reevaluateSubmissionRequest struct {
		SubmUUIDs []string `json:"subm_uuids"`
	}

	var request reevaluateSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subms := []*submsrvc.Submission{}
	for _, submUuid := range request.SubmUUIDs {
		subm, err := httpserver.submSrvc.ReevaluateSubmission(r.Context(), submUuid)
		if err != nil {
			handleJsonSrvcError(logger, w, err)
			return
		}
		subms = append(subms, subm)
	}

	briefSubms := make([]*BriefSubmission, len(subms))
	for i, subm := range subms {
		briefSubms[i] = mapBriefSubm(subm)
	}
	response := briefSubms

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
