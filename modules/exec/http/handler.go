package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/modules/exec"
	"github.com/programme-lv/backend/modules/user/auth"
)

type ExecHttpHandler struct {
	execSrvc    exec.CodeExecutionService
	adminAPIKey []byte
}

func NewExecHttpHandler(execSrvc exec.CodeExecutionService, adminAPIKey []byte) *ExecHttpHandler {
	return &ExecHttpHandler{
		execSrvc:    execSrvc,
		adminAPIKey: adminAPIKey,
	}
}

func (h *ExecHttpHandler) RegisterRoutes(r *chi.Mux) {
	r.Group(func(r chi.Router) {
		r.Use(auth.HttpAllowOnlyAdmins(h.adminAPIKey))
		r.Post("/tester/run", h.testerRun)
		r.Get("/tester/run/{evalUuid}", h.testerListen)
		r.Get("/exec/{execUuid}", h.execGet)
	})
}
