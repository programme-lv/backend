package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/user"
	"github.com/programme-lv/backend/modules/user/auth"
)

type User struct {
	UUID          string  `json:"uuid"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	Firstname     *string `json:"firstname"`
	Lastname      *string `json:"lastname"`
	EmailVerified bool    `json:"email_verified"`
}

func toHTTPUser(u *user.User) User {
	return User{
		UUID:          u.UUID.String(),
		Username:      u.Username,
		Email:         u.Email,
		Firstname:     u.Firstname,
		Lastname:      u.Lastname,
		EmailVerified: u.EmailVerified,
	}
}

func toHTTPUserValue(u user.User) User {
	return toHTTPUser(&u)
}

func requireUserUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil {
		jsonresp.Unauthorized(w, "authentication required")
		return uuid.Nil, false
	}
	userUUID, err := uuid.Parse(claims.UUID)
	if err != nil {
		jsonresp.Unauthorized(w, "authentication required")
		return uuid.Nil, false
	}
	return userUUID, true
}
