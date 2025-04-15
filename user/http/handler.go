package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/user"
	"github.com/programme-lv/backend/user/auth"
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
	r.Group(func(r chi.Router) {
		r.Use(auth.GetJwtAuthMiddleware(h.JwtKey))
		r.Post("/login", h.Login)
		r.Post("/users", h.Register)
		r.Get("/role", h.GetRole)
	})
}
