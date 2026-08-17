package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/subm/srvc"
	"github.com/programme-lv/backend/modules/user/auth"
)

var ErrSubmissionTooFrequent = srvcerror.New(
	"submission_too_frequent",
	"Uzgaidiet pirms nākamā iesūtījuma!",
).SetHttpStatusCode(http.StatusTooManyRequests)

func errSubmissionTooFrequent(delaySeconds int) srvcerror.E {
	return ErrSubmissionTooFrequent.WithMsg(
		fmt.Sprintf("Uzgaidiet %d sekundes pirms nākamā iesūtījuma!", delaySeconds),
	)
}

var ErrUnauthorized = srvcerror.New(
	"unauthorized_access",
	"nav autorizācijas",
).SetHttpStatusCode(http.StatusUnauthorized)

var ErrUnauthorizedUsernameMismatch = ErrUnauthorized.WithMsg(
	"JWT norādītais lietotājvārds nesakrīt ar pieprasīto lietotājvārdu",
)

var ErrJwtTokenMissing = ErrUnauthorized.WithMsg("JWT netika atrasts")

func (h *SubmHttpHandler) PostSubm(w http.ResponseWriter, r *http.Request) {
	log := ctxlog.FromContext(r.Context())
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
		jsonresp.HandleErrorWithContext(r.Context(), w, ErrJwtTokenMissing)
		return
	}

	var request createSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("decode request body", "error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if claims.Username != request.Username {
		log.Warn("unauthorized username mismatch", "jwt_username", claims.Username, "request_username", request.Username)
		jsonresp.HandleErrorWithContext(r.Context(), w, ErrUnauthorizedUsernameMismatch)
		return
	}

	h.rateLock.Lock()
	lastTime, exists := h.lastSubmTime[request.Username]
	now := time.Now()
	if exists && now.Sub(lastTime) < 10*time.Second {
		h.rateLock.Unlock()
		log.Warn("submission too frequent", "username", request.Username, "last_time", lastTime)
		jsonresp.HandleErrorWithContext(r.Context(), w, errSubmissionTooFrequent(10))
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
		jsonresp.HandleErrorWithContext(r.Context(), w, err)
		return
	}

	submUUID := uuid.New()
	log.Info("generated submission UUID", "subm_uuid", submUUID)

	submitErr := h.submSrvc.SubmitSol(r.Context(), srvc.SubmitSolParams{
		UUID:        submUUID,
		Submission:  request.Submission,
		AuthorUUID:  author.UUID,
		ProgrLangID: request.ProgrammingLangID,
		TaskShortID: request.TaskCodeID,
	})
	if submitErr != nil {
		jsonresp.HandleErrorWithContext(r.Context(), w, submitErr)
		return
	}

	h.submCache.Flush()

	subm, viewSubmErr := h.submSrvc.ViewSubm(r.Context(), submUUID)
	if viewSubmErr != nil {
		jsonresp.HandleErrorWithContext(r.Context(), w, viewSubmErr)
		return
	}

	response, mapSubmErr := h.mapSubm(r.Context(), subm)
	if mapSubmErr != nil {
		jsonresp.HandleErrorWithContext(r.Context(), w, mapSubmErr)
		return
	}

	log.Info("submission created successfully", "subm_uuid", submUUID)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
