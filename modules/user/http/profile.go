package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/jsonresp"
)

func (h *UserHttpHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := requireUserUUID(w, r)
	if !ok {
		return
	}

	var request struct {
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	user, err := h.userSrvc.UpdateProfile(r.Context(), userUUID, request.Firstname, request.Lastname)
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	jsonresp.Success(w, toHTTPUser(user))
}
