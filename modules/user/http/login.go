package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/common/srvcerror"
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

	user, loginErr := httpserver.userSrvc.Login(r.Context(), request.Username, request.Password)
	if loginErr != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, loginErr)
		return
	}

	if cookieErr := httpserver.issueAuthCookie(w, user); cookieErr != nil {
		slog.Error("generate JWT", "error", cookieErr)
		jsonresp.HandleSrvcError(slog.Default(), w, srvcerror.InternalServerError())
		return
	}

	jsonresp.Success(w, toHTTPUser(user))
}
