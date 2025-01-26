package submhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/subm/submcmds"
	"github.com/programme-lv/backend/subm/submqueries"
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

	submUUID := uuid.New()

	err := h.submSrvc.CreateSubm.Handle(r.Context(), submcmds.CreateSubmParams{
		UUID:        submUUID,
		Submission:  request.Submission,
		Username:    request.Username,
		ProgrLangID: request.ProgrammingLangID,
		TaskShortID: request.TaskCodeID,
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	err = h.submSrvc.CreateEval.Handle(r.Context(), submcmds.CreateEvalParams{
		EvalUUID: uuid.New(),
		SubmUUID: submUUID,
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	err = h.submSrvc.AttachEval.Handle(r.Context(), submcmds.AttachEvalParams{
		EvalUUID: uuid.New(),
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	err = h.submSrvc.EnqueueEval.Handle(r.Context(), submcmds.EnqueueEvalParams{
		EvalUUID: uuid.New(),
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	subm, err := h.submSrvc.GetSubm.Handle(r.Context(), submqueries.GetSubmParams{
		SubmUUID: submUUID,
	})
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
