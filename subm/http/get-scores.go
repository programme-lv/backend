package http

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
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	scores, err := h.submSrvc.GetMaxScorePerTask(r.Context(), user.UUID)
	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	scoresJson := make(map[string]MaxScore)
	for taskId, score := range scores {
		scoresJson[taskId], err = h.mapMaxScore(r.Context(), taskId, score)
		if err != nil {
			httpjson.HandleSrvcError(slog.Default(), w, err)
			return
		}
	}

	httpjson.WriteSuccessJson(w, scoresJson)
}
