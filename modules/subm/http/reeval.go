package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/jsonresp"
)

func (h *SubmHttpHandler) ReevalSubms(w http.ResponseWriter, r *http.Request) {
	l := h.newLogger(r.Context())

	type reevalSubmsRequest struct {
		SubmUUIDs []uuid.UUID `json:"subm_uuids"`
	}

	var request reevalSubmsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		jsonresp.BadRequest(w, "nederīgs JSON")
		return
	}

	if len(request.SubmUUIDs) == 0 {
		jsonresp.BadRequest(w, "no subm uuids provided")
		return
	}

	for _, uuid := range request.SubmUUIDs {
		if err := h.submSrvc.ReEvalSubm(r.Context(), uuid); err != nil {
			jsonresp.HandleSrvcError(l, w, err)
			return
		}
	}

	jsonresp.Success(w, "reevaluation enqueued for all provided submissions")
}
