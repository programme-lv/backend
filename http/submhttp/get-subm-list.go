package submhttp

import (
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submqueries"
)

func (h *SubmHttpHandler) GetSubmList(w http.ResponseWriter, r *http.Request) {
	subms, err := h.submSrvc.ListSubmsQuery.Handle(r.Context(), submqueries.ListSubmsParams{
		Limit:  30,
		Offset: 0,
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	mapSubmList := func(subms []subm.Subm) []SubmListEntry {
		response := make([]SubmListEntry, len(subms))
		for i, subm := range subms {
			response[i] = h.mapSubmListEntry(r.Context(), subm)
		}
		return response
	}

	response := mapSubmList(subms)

	httpjson.WriteSuccessJson(w, response)
}
