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

var ErrJwtTokenMissing = ErrUnauthorized.WithMsg("JWT netika atrasts")

func (h *SubmHttpHandler) PostSubm(w http.ResponseWriter, r *http.Request) {
	log := ctxlog.FromContext(r.Context())

	type createSubmissionRequest struct {
		Submission        string `json:"submission"`
		ProgrammingLangID string `json:"programming_lang_id"`
		TaskCodeID        string `json:"task_code_id"`
	}

	authorUUID, err := auth.GetUserUuidFromCtx(r.Context())
	if err != nil {
		log.Warn("JWT user UUID missing or invalid")
		jsonresp.HandleErrorWithContext(r.Context(), w, ErrJwtTokenMissing)
		return
	}

	var request createSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("decode request body", "error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	h.rateLock.Lock()
	lastTime, exists := h.lastSubmTime[authorUUID]
	now := time.Now()
	if exists && now.Sub(lastTime) < 10*time.Second {
		h.rateLock.Unlock()
		log.Warn("submission too frequent", "author_uuid", authorUUID, "last_time", lastTime)
		jsonresp.HandleErrorWithContext(r.Context(), w, errSubmissionTooFrequent(10))
		return
	}
	h.lastSubmTime[authorUUID] = now
	h.rateLock.Unlock()

	submUUID := uuid.New()

	submitErr := h.submSrvc.SubmitSol(r.Context(), srvc.SubmitSolParams{
		UUID:        submUUID,
		Submission:  request.Submission,
		AuthorUUID:  authorUUID,
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

	log.Debug(
		"submission created",
		"subm_uuid", submUUID,
		"author_uuid", authorUUID,
		"programming_lang_id", request.ProgrammingLangID,
		"task_code_id", request.TaskCodeID,
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("write submission response", "error", err)
	}
}
