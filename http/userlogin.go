package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/user"
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

	token, err := httpserver.userSrvc.Login(context.TODO(), &user.LoginPayload{
		Username: request.Username,
		Password: request.Password,
	})

	if err != nil {
		userErr := &user.Error{}
		if errors.As(err, &userErr) {
			writeJsonErrorResponse(w, userErr.Error(), userErr.HttpStatusCode(), userErr.ErrorCode())
			return
		}
		logger.Error("internal server error", "error", err)
		writeJsonInternalServerError(w)
		return
	}

	writeJsonSuccessResponse(w, token)
}
