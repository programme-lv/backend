package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/programme-lv/backend/subm"
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

	// TODO: JWT authentification
	// TODO: make JWT claims an explicit parameter

	subm, err := httpserver.submSrvc.CreateSubmission(context.TODO(), &subm.CreateSubmissionPayload{
		Submission:        request.Submission,
		Username:          request.Username,
		ProgrammingLangID: request.ProgrammingLangID,
		TaskCodeID:        request.TaskCodeID,
		Token:             request.Token,
	})

	if err != nil {
		// TODO: better error handling
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response := mapSubmissionResponse(subm)

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
