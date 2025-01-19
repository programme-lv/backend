package submhttp

import (
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submqueries"
)

func (h *SubmHttpServer) GetSubmList(w http.ResponseWriter, r *http.Request) {
	subms, err := h.submSrvc.ListSubmsQuery.Handle(r.Context(), submqueries.ListSubmsParams{
		Limit:  30,
		Offset: 0,
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	mapSubmList := func(subms []subm.Subm) []Subm {
		response := make([]Subm, len(subms))
		for i, subm := range subms {
			ptr, err := h.mapSubm(r.Context(), subm)
			if err != nil {
				httpjson.HandleError(slog.Default(), w, err)
				return nil
			}
			response[i] = *ptr
			response[i].Content = "" // don't send content to list view
		}
		return response
	}

	response := mapSubmList(subms)

	httpjson.WriteSuccessJson(w, response)
}
