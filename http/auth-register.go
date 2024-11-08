package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/user"
)

func (httpserver *HttpServer) authRegister(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type registerRequest struct {
		Username  string  `json:"username"`
		Email     string  `json:"email"`
		Firstname *string `json:"firstname"`
		Lastname  *string `json:"lastname"`
		Password  string  `json:"password"`
	}

	type registerResponse struct {
		UUID      string  `json:"uuid"`
		Username  string  `json:"username"`
		Email     string  `json:"email"`
		Firstname *string `json:"firstname"`
		Lastname  *string `json:"lastname"`
	}

	var request registerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	user, err := httpserver.userSrvc.CreateUser(context.TODO(), &user.UserPayload{
		Username:  request.Username,
		Email:     request.Email,
		Firstname: request.Firstname,
		Lastname:  request.Lastname,
		Password:  request.Password,
	})

	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	response := registerResponse{
		UUID:      user.UUID.String(),
		Username:  user.Username,
		Email:     user.Email,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
	}

	writeJsonSuccessResponse(w, response)
}
