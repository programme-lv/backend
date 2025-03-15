package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/logger"
)

func (h *SubmHttpHandler) GetFullSubm(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	submUuidStr := chi.URLParam(r, "subm-uuid")
	log.Info("getting full submission", "subm_uuid", submUuidStr)

	submUuid, err := uuid.Parse(submUuidStr)
	if err != nil {
		log.Warn("invalid submission UUID", "subm_uuid", submUuidStr, "error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subm, err := h.submSrvc.GetSubm(r.Context(), submUuid)
	if err != nil {
		log.Error("failed to get submission", "subm_uuid", submUuid, "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	response, err := h.mapSubm(r.Context(), subm)
	if err != nil {
		log.Error("failed to map submission", "subm_uuid", submUuid, "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	log.Info("returning full submission", "subm_uuid", submUuid)
	httpjson.WriteSuccessJson(w, response)
}
