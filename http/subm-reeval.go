package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/srvcerror"
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

	for _, submUuid := range request.SubmUUIDs {
		submUuid, err := uuid.Parse(submUuid)
		if err != nil {
			format := "failed to parse submission UUID: %w"
			errMsg := fmt.Errorf(format, err)
			handleJsonSrvcError(logger, w, srvcerror.ErrInternalSE().SetDebug(errMsg))
			return
		}
		err = httpserver.submSrvc.ReevaluateSubmission(r.Context(), submUuid)
		if err != nil {
			handleJsonSrvcError(logger, w, err)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
}
