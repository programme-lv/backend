package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/auth"
	"github.com/programme-lv/backend/usersrvc"
)

func (httpserver *HttpServer) authLogin(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type loginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var request loginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	logger.Info("received login request", "username", request.Username)

	user, err := httpserver.userSrvc.Login(context.TODO(), &usersrvc.LoginParams{
		Username: request.Username,
		Password: request.Password,
	})
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	token, err := auth.GenerateJWT(
		user.Username,
		user.Email, user.UUID,
		user.Firstname, user.Lastname,
		httpserver.JwtKey)
	if err != nil {
		logger := httplog.LogEntry(r.Context())
		logger.Error("failed to generate JWT", "error", err)
		writeJsonInternalServerError(w)
		return
	}

	writeJsonSuccessResponse(w, token)
}
