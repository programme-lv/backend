package submhttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/httpjson"
)

func (h *SubmHttpHandler) GetFullSubm(w http.ResponseWriter, r *http.Request) {
	submUuidStr := chi.URLParam(r, "subm-uuid")
	submUuid, err := uuid.Parse(submUuidStr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subm, err := h.submSrvc.GetSubm(r.Context(), submUuid)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	response, err := h.mapSubm(r.Context(), subm)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	httpjson.WriteSuccessJson(w, response)
}
