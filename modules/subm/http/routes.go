package http

import (
	"github.com/go-chi/chi/v5"
	hf "github.com/programme-lv/backend/common/httpfunc"
	"github.com/programme-lv/backend/modules/user/auth"
)

func (h *SubmHttpHandler) RegisterRoutes(r *chi.Mux, jwtKey, adminAPIKey []byte, cookieSecure bool, pwdChangedAt auth.PasswordChangedAtLookup) {
	r.Group(func(r chi.Router) {
		r.Use(auth.HttpJwtAuthentication(
			jwtKey,
			auth.WithSecureCookie(cookieSecure),
			auth.WithPasswordChangedAtLookup(pwdChangedAt),
		))
		r.Post("/subm", h.PostSubm)
		r.Get("/subm", h.GetSubmList)
		r.Get("/subm/{subm-id}", h.GetFullSubm)
		r.Get("/subm/scores/{username}", h.GetMaxScorePerTask)
		r.Get("/subm-updates", h.ListenToSubmListUpdates)

		// admin-only routes
		r.Group(func(r chi.Router) {
			r.Use(auth.HttpAllowOnlyAdmins(adminAPIKey))
			r.Post("/reeval", h.ReevalSubms)

			r.Get("/tasks/{taskId}/solutions", hf.NoReqJsonResp(h.ListTaskSolutions))
			r.Get("/tasks/{taskId}/solutions/calibration", hf.NoReqJsonResp(h.GetTaskCalibration))
			r.Post("/tasks/{taskId}/solutions/{fname}/profile", hf.JsonReqJsonResp(h.StartSolutionProfile))
			r.Get("/tasks/{taskId}/solutions/{fname}/profiles", hf.NoReqJsonResp(h.ListSolutionProfiles))
			r.Get("/solution-profiles/{profileId}", hf.NoReqJsonResp(h.GetSolutionProfile))
		})
	})
}
