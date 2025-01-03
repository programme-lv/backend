package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

func (httpserver *HttpServer) reevaluateSubmissions(w http.ResponseWriter, r *http.Request) {
	type reevaluateSubmissionsRequest struct {
		SubmUUIDs []string `json:"subm_uuids"`
	}

	var request reevaluateSubmissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	for _, submUuid := range request.SubmUUIDs {
		uuid, err := uuid.Parse(submUuid)
		if err != nil {
			handleJsonSrvcError(slog.Default(), w, err)
			return
		}
		err = httpserver.submSrvc.ReevalSubm(r.Context(), uuid)
		if err != nil {
			handleJsonSrvcError(slog.Default(), w, err)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}
