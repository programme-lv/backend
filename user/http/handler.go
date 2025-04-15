package http

import (
	"github.com/programme-lv/backend/user"
)

type UserHttpHandler struct {
	userSrvc *user.UserSrvc
	JwtKey   []byte
}

func NewUserHttpHandler(userSrvc *user.UserSrvc, jwtKey []byte) *UserHttpHandler {
	return &UserHttpHandler{
		userSrvc: userSrvc,
		JwtKey:   jwtKey,
	}
}
