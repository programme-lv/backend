package http

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
)

// scoresUsername returns the {username} path parameter decoded exactly once.
//
// chi routes on URL.RawPath when it is set and on URL.Path otherwise, and
// chi.URLParam returns that same representation. RawPath still contains
// percent escapes, while Path is already decoded. Decoding unconditionally
// would therefore turn a literal "%40" username (sent as "%2540") into "@".
func scoresUsername(r *http.Request) (string, error) {
	username := chi.URLParam(r, "username")
	if r.URL.RawPath == "" {
		return username, nil
	}
	return url.PathUnescape(username)
}

func (h *SubmHttpHandler) GetMaxScorePerTask(w http.ResponseWriter, r *http.Request) {
	username, usernameErr := scoresUsername(r)
	if usernameErr != nil {
		jsonresp.WriteError(w, jsonresp.ErrHttpBadRequest)
		return
	}

	user, err := h.userSrvc.GetUserByUsername(r.Context(), username)
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	scores, getScoresErr := h.submSrvc.GetMaxScorePerTask(r.Context(), user.UUID)
	if getScoresErr != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, getScoresErr)
		return
	}

	scoresJson := make(map[string]MaxScore)
	for taskId, score := range scores {
		var mapMaxScoreErr error
		scoresJson[taskId], mapMaxScoreErr = h.mapMaxScore(r.Context(), taskId, score)
		if mapMaxScoreErr != nil {
			jsonresp.HandleSrvcError(slog.Default(), w, mapMaxScoreErr)
			return
		}
	}

	jsonresp.Success(w, scoresJson)
}
