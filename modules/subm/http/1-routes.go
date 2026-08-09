package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/modules/user/auth"
)

func (h *SubmHttpHandler) RegisterRoutes(r *chi.Mux, jwtKey, adminAPIKey []byte) {
	r.Group(func(r chi.Router) {
		r.Use(auth.HttpJwtAuthentication(jwtKey))
		r.Post("/subm", h.PostSubm)
		r.Get("/subm", h.GetSubmList)
		r.Get("/subm/{subm-uuid}", h.GetFullSubm)
		r.Get("/subm/scores/{username}", h.GetMaxScorePerTask)
		r.Get("/subm-updates", h.ListenToSubmListUpdates)

		// admin-only routes
		r.Group(func(r chi.Router) {
			r.Use(auth.HttpAllowOnlyAdmins(adminAPIKey))
			r.Post("/reeval", h.ReevalSubms)
		})
	})
}
