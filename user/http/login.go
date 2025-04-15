package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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

	validFor := 24 * time.Hour

	token, err := auth.GenerateJWT(
		user.Username,
		user.Email, user.UUID,
		user.Firstname, user.Lastname,
		httpserver.JwtKey, validFor)
	if err != nil {
		err = fmt.Errorf("failed to generate JWT: %w", err)
		httpjson.HandleError(slog.Default(), w, err)
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
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil, // Set Secure flag if using HTTPS
	}
	http.SetCookie(w, &cookie)

	// Send success response without including the token in the body
	httpjson.WriteSuccessJson(w, map[string]string{"message": "Login successful"})
}
