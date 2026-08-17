package http

import (
	"net/http"
	"time"

	"github.com/programme-lv/backend/modules/user"
	"github.com/programme-lv/backend/modules/user/auth"
)

func (h *UserHttpHandler) issueAuthCookie(w http.ResponseWriter, u *user.User) error {
	validFor := 24 * time.Hour
	token, err := auth.GenerateJWT(u.Username, u.Email, u.UUID, h.jwtKey, validFor)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(validFor),
		HttpOnly: true,
		Path:     "/",
		Domain:   h.cookieDomain,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieSecure,
	})
	return nil
}
