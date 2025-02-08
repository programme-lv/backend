package submhttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/httpjson"
)

func (h *SubmHttpHandler) GetMaxScorePerTask(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	user, err := h.userSrvc.GetUserByUsername(r.Context(), username)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	scores, err := h.submSrvc.GetMaxScorePerTask(r.Context(), user.UUID)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	httpjson.WriteSuccessJson(w, scores)
}
