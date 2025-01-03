package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/submsrvc"
)

func (httpserver *HttpServer) createSubmission(w http.ResponseWriter, r *http.Request) {
	type createSubmissionRequest struct {
		Submission        string `json:"submission"`
		Username          string `json:"username"`
		ProgrammingLangID string `json:"programming_lang_id"`
		TaskCodeID        string `json:"task_code_id"`
		Token             string `json:"token"`
	}

	var request createSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subm, err := httpserver.submSrvc.CreateSubmission(r.Context(), &submsrvc.CreateSubmissionParams{
		Submission: request.Submission,
		Username:   request.Username,
		ProgLangID: request.ProgrammingLangID,
		TaskCodeID: request.TaskCodeID,
	})

	if err != nil {
		handleJsonSrvcError(slog.Default(), w, err)
		return
	}

	response := mapSubm(*subm)

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
