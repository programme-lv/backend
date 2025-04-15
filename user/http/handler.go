package http

import (
	"github.com/go-chi/chi/v5"
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

func (h *UserHttpHandler) RegisterRoutes(r *chi.Mux) {
	r.Post("/login", h.Login)
	r.Post("/users", h.Register)
}
