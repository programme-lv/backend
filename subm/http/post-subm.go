package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/auth"
	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/logger"
	"github.com/programme-lv/backend/subm/submerror"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
)

func (h *SubmHttpHandler) PostSubm(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Info("processing submission request")

	type createSubmissionRequest struct {
		Submission        string `json:"submission"`
		Username          string `json:"username"`
		ProgrammingLangID string `json:"programming_lang_id"`
		TaskCodeID        string `json:"task_code_id"`
	}

	claims := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if claims == nil {
		log.Warn("JWT token missing")
		httpjson.HandleErrorWithContext(*r, w, submerror.ErrJwtTokenMissing())
		return
	}

	var request createSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("failed to decode request body", "error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if claims.Username != request.Username {
		log.Warn("unauthorized username mismatch", "jwt_username", claims.Username, "request_username", request.Username)
		httpjson.HandleErrorWithContext(*r, w, submerror.ErrUnauthorizedUsernameMismatch())
		return
	}

	// Check submission rate limit (at least 10 seconds between submissions)
	h.rateLock.Lock()
	lastTime, exists := h.lastSubmTime[request.Username]
	now := time.Now()
	if exists && now.Sub(lastTime) < 10*time.Second {
		h.rateLock.Unlock()
		log.Warn("submission too frequent", "username", request.Username, "last_time", lastTime)
		httpjson.HandleErrorWithContext(*r, w, submerror.ErrSubmissionTooFrequent(10))
		return
	}
	h.lastSubmTime[request.Username] = now
	h.rateLock.Unlock()

	log.Info(
		"post submission request details",
		"username", request.Username,
		"programming_lang_id", request.ProgrammingLangID,
		"task_code_id", request.TaskCodeID,
	)

	author, err := h.userSrvc.GetUserByUsername(r.Context(), request.Username)
	if err != nil {
		log.Error("failed to get user by username", "username", request.Username, "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	submUUID := uuid.New()
	log.Info("generated submission UUID", "subm_uuid", submUUID)

	err = h.submSrvc.SubmitSol(r.Context(), submcmd.SubmitSolParams{
		UUID:        submUUID,
		Submission:  request.Submission,
		AuthorUUID:  author.UUID,
		ProgrLangID: request.ProgrammingLangID,
		TaskShortID: request.TaskCodeID,
	})
	if err != nil {
		log.Error("failed to submit solution", "subm_uuid", submUUID, "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	subm, err := h.submSrvc.GetSubm(r.Context(), submUUID)
	if err != nil {
		log.Error("failed to get submission after creation", "subm_uuid", submUUID, "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	response, err := h.mapSubm(r.Context(), subm)
	if err != nil {
		log.Error("failed to map submission", "subm_uuid", submUUID, "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	log.Info("submission created successfully", "subm_uuid", submUUID)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
