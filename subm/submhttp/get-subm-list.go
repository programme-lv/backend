package submhttp

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
)

func (h *SubmHttpHandler) GetSubmList(w http.ResponseWriter, r *http.Request) {
	subms, err := h.submSrvc.ListSubms(r.Context(), submquery.ListSubmsParams{
		Limit:  30,
		Offset: 0,
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	mapSubmList := func(subms []subm.Subm) []SubmListEntry {
		response := make([]SubmListEntry, 0)
		for _, subm := range subms {
			log.Printf("subm: %+v", subm.UUID)
			entry, err := h.mapSubmListEntry(r.Context(), subm)
			if err != nil {
				slog.Default().Warn("failed to map subm list entry", "error", err)
				continue
			}
			response = append(response, entry)
		}
		return response
	}

	response := mapSubmList(subms)

	httpjson.WriteSuccessJson(w, response)
}
