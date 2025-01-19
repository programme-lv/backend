package submhttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/subm/submqueries"
)

func (h *SubmHttpServer) getSubm(w http.ResponseWriter, r *http.Request) {
	submUuidStr := chi.URLParam(r, "submUuid")
	submUuid, err := uuid.Parse(submUuidStr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subm, err := h.submSrvc.GetSubmQuery.Handle(r.Context(), submqueries.GetSubmParams{
		SubmUUID: submUuid,
	})
	if err != nil {
		handleJsonSrvcError(slog.Default(), w, err)
		return
	}

	response := mapSubm(*subm)

	writeJsonSuccessResponse(w, response)
}
