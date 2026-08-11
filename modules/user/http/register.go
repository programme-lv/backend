package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/user"
)

func (httpserver *UserHttpHandler) Register(w http.ResponseWriter, r *http.Request) {
	type registerRequest struct {
		Username  string  `json:"username"`
		Email     string  `json:"email"`
		Firstname *string `json:"firstname"`
		Lastname  *string `json:"lastname"`
		Password  string  `json:"password"`
	}

	var request registerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	user, err := httpserver.userSrvc.CreateUser(context.TODO(), user.CreateUserParams{
		Username:  request.Username,
		Email:     request.Email,
		Firstname: request.Firstname,
		Lastname:  request.Lastname,
		Password:  request.Password,
	})

	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	jsonresp.Success(w, toHTTPUser(user))
}
