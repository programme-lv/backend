package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/user/auth"
)

func (httpserver *UserHttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	type loginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var request loginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	user, err := httpserver.userSrvc.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	token, err := auth.GenerateJWT(
		user.Username,
		user.Email, user.UUID,
		user.Firstname, user.Lastname,
		httpserver.JwtKey)
	if err != nil {
		err = fmt.Errorf("failed to generate JWT: %w", err)
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	httpjson.WriteSuccessJson(w, token)
}
