package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/common/srvcerror"
)

func (h *UserHttpHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := requireUserUUID(w, r)
	if !ok {
		return
	}

	var request struct {
		CurrentPassword string `json:"current_password"`
		Password        string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.userSrvc.ChangePassword(r.Context(), userUUID, request.CurrentPassword, request.Password); err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	user, getErr := h.userSrvc.GetUserByUUID(r.Context(), userUUID)
	if getErr != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, getErr)
		return
	}

	if cookieErr := h.issueAuthCookie(w, &user); cookieErr != nil {
		slog.Error("generate JWT", "error", cookieErr)
		jsonresp.HandleSrvcError(slog.Default(), w, srvcerror.InternalServerError())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
