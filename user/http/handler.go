package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/user"
	"github.com/programme-lv/backend/user/auth"
)

type UserHttpHandler struct {
	userSrvc     *user.UserSrvc
	jwtKey       []byte
	cookieDomain string
}

// NewUserHttpHandler creates a new UserHttpHandler with the given user service and JWT key.
// The cookieDomain parameter is optional and defaults to an empty string if not provided.
func NewUserHttpHandler(userSrvc *user.UserSrvc, jwtKey []byte, options ...func(*UserHttpHandler)) *UserHttpHandler {
	handler := &UserHttpHandler{
		userSrvc:     userSrvc,
		jwtKey:       jwtKey,
		cookieDomain: "",
	}

	// Apply any provided options
	for _, option := range options {
		option(handler)
	}

	return handler
}

// WithCookieDomain sets the cookie domain for the UserHttpHandler
func WithCookieDomain(domain string) func(*UserHttpHandler) {
	return func(h *UserHttpHandler) {
		h.cookieDomain = domain
	}
}

func (h *UserHttpHandler) RegisterRoutes(r *chi.Mux) {
	r.Group(func(r chi.Router) {
		r.Use(auth.GetJwtAuthMiddleware(h.jwtKey))
		r.Post("/login", h.Login)
		r.Post("/users", h.Register)
		r.Get("/role", h.GetRole)
		r.Post("/logout", h.Logout)
		r.Get("/whoami", h.WhoAmI)
	})
}
