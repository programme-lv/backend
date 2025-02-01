package submhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
)

func (h *SubmHttpHandler) PostSubm(w http.ResponseWriter, r *http.Request) {
	type createSubmissionRequest struct {
		Submission        string `json:"submission"`
		Username          string `json:"username"`
		ProgrammingLangID string `json:"programming_lang_id"`
		TaskCodeID        string `json:"task_code_id"`
	}

	var request createSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	slog.Default().Info(
		"post subm request",
		"username",
		request.Username,
		"programming_lang_id",
		request.ProgrammingLangID,
		"task_code_id",
		request.TaskCodeID,
	)

	author, err := h.userSrvc.GetUserByUsername(r.Context(), request.Username)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	submUUID := uuid.New()

	err = h.submSrvc.SubmitSol(r.Context(), submcmd.SubmitSolParams{
		UUID:        submUUID,
		Submission:  request.Submission,
		AuthorUUID:  author.UUID,
		ProgrLangID: request.ProgrammingLangID,
		TaskShortID: request.TaskCodeID,
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	subm, err := h.submSrvc.GetSubm(r.Context(), submUUID)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	response, err := h.mapSubm(r.Context(), subm)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
