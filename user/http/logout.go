package http

import (
	"net/http"

	"github.com/programme-lv/backend/httpjson"
)

// Logout handles user logout by clearing the auth_token cookie
func (httpserver *UserHttpHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear the auth token cookie
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
		Secure:   r.TLS != nil,
	}
	http.SetCookie(w, &cookie)

	httpjson.WriteSuccessJson(w, map[string]string{"message": "Logout successful"})
}
