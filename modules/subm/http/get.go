package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/subm/domain"
)

type submPathID struct {
	UUID    uuid.UUID
	ShortID string
}

func parseSubmPathID(s string) (submPathID, bool) {
	if u, err := uuid.Parse(s); err == nil {
		return submPathID{UUID: u}, true
	}
	if domain.ValidShortID(s) {
		return submPathID{ShortID: s}, true
	}
	return submPathID{}, false
}

func (h *SubmHttpHandler) GetFullSubm(w http.ResponseWriter, r *http.Request) {
	log := ctxlog.FromContext(r.Context())

	id := chi.URLParam(r, "subm-id")
	log.Info("getting full submission", "subm_id", id)

	parsed, ok := parseSubmPathID(id)
	if !ok {
		log.Warn("invalid submission id", "subm_id", id)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var subm domain.Subm
	if parsed.UUID != uuid.Nil {
		viewed, err := h.submSrvc.ViewSubm(r.Context(), parsed.UUID)
		if err != nil {
			jsonresp.HandleErrorWithContext(r.Context(), w, err)
			return
		}
		subm = viewed
	} else {
		viewed, err := h.submSrvc.ViewSubmByShortID(r.Context(), parsed.ShortID)
		if err != nil {
			jsonresp.HandleErrorWithContext(r.Context(), w, err)
			return
		}
		subm = viewed
	}

	response, mapErr := h.mapSubm(r.Context(), subm)
	if mapErr != nil {
		jsonresp.HandleErrorWithContext(r.Context(), w, mapErr)
		return
	}

	log.Info("returning full submission", "subm_uuid", subm.UUID, "short_id", subm.ShortID)
	jsonresp.Success(w, response)
}
