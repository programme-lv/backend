package http

import (
	"net/http"

	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/logger"
	"github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
)

func (h *SubmHttpHandler) GetSubmList(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Info("getting submission list")

	subms, err := h.submSrvc.ListSubms(r.Context(), submquery.ListSubmsParams{
		Limit:  30,
		Offset: 0,
	})
	if err != nil {
		log.Error("failed to list submissions", "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	log.Debug("submissions retrieved successfully", "count", len(subms))

	mapSubmList := func(subms []domain.Subm) []SubmListEntry {
		response := make([]SubmListEntry, 0)
		for _, subm := range subms {
			entry, err := h.mapSubmListEntry(r.Context(), subm)
			if err != nil {
				log.Warn("failed to map subm list entry", "error", err)
				continue
			}
			response = append(response, entry)
		}
		return response
	}

	response := mapSubmList(subms)
	log.Info("returning submission list", "count", len(response))

	httpjson.WriteSuccessJson(w, response)
}
