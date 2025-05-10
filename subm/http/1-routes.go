package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/user/auth"
)

func (h *SubmHttpHandler) RegisterRoutes(r *chi.Mux, jwtKey []byte) {
	r.Group(func(r chi.Router) {
		r.Use(auth.GetJwtAuthMiddleware(jwtKey))
		r.Post("/subm", h.PostSubm)
		r.Get("/subm", h.GetSubmList)
		r.Get("/subm/{subm-uuid}", h.GetFullSubm)
		r.Get("/subm/scores/{username}", h.GetMaxScorePerTask)

		// Admin-only routes
		r.Group(func(r chi.Router) {
			r.Use(auth.AdminOnly)
			r.Post("/reeval", h.ReevalSubms)
		})
	})
}
