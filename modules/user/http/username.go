package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/jsonresp"
)

func (h *UserHttpHandler) ChangeUsername(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := requireUserUUID(w, r)
	if !ok {
		return
	}
	var request struct {
		Username        *string `json:"username"`
		CurrentPassword *string `json:"current_password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.Username == nil || request.CurrentPassword == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	username, err := h.userSrvc.ChangeUsername(r.Context(), userUUID, *request.Username, *request.CurrentPassword)
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}
	jsonresp.Success(w, struct {
		Username string `json:"username"`
	}{username})
}
