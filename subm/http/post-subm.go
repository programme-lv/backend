package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/auth"
	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/subm/submerror"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
)

func (h *SubmHttpHandler) PostSubm(w http.ResponseWriter, r *http.Request) {
	type createSubmissionRequest struct {
		Submission        string `json:"submission"`
		Username          string `json:"username"`
		ProgrammingLangID string `json:"programming_lang_id"`
		TaskCodeID        string `json:"task_code_id"`
	}

	claims := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if claims == nil {
		httpjson.HandleError(slog.Default(), w, submerror.ErrJwtTokenMissing())
		return
	}

	var request createSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if claims.Username != request.Username {
		httpjson.HandleError(slog.Default(), w, submerror.ErrUnauthorizedUsernameMismatch())
		return
	}

	// Check submission rate limit (at least 10 seconds between submissions)
	h.rateLock.Lock()
	lastTime, exists := h.lastSubmTime[request.Username]
	now := time.Now()
	if exists && now.Sub(lastTime) < 10*time.Second {
		h.rateLock.Unlock()
		httpjson.HandleError(slog.Default(), w, submerror.ErrSubmissionTooFrequent(10))
		return
	}
	h.lastSubmTime[request.Username] = now
	h.rateLock.Unlock()

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
