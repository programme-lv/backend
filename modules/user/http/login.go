package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/user/auth"
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

	validFor := 24 * time.Hour

	token, jwtErr := auth.GenerateJWT(
		user.Username,
		user.Email, user.UUID,
		httpserver.jwtKey, validFor)
	if jwtErr != nil {
		slog.Error("generate JWT", "error", jwtErr)
		jsonresp.HandleSrvcError(slog.Default(), w, srvcerror.InternalServerError())
		return
	}

	// Set the JWT token as HTTP-only cookie
	expirationTime := time.Now().Add(validFor)
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  expirationTime,
		HttpOnly: true,
		Path:     "/",
		Domain:   httpserver.cookieDomain,
		SameSite: http.SameSiteLaxMode,
		Secure:   httpserver.cookieSecure,
	}
	http.SetCookie(w, &cookie)

	jsonresp.Success(w, toHTTPUser(user))
}
