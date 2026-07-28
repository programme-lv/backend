package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/modules/exec"
)

type ExecHttpHandler struct {
	execSrvc exec.CodeExecutionService
}

func NewExecHttpHandler(execSrvc exec.CodeExecutionService) *ExecHttpHandler {
	return &ExecHttpHandler{
		execSrvc: execSrvc,
	}
}

func (h *ExecHttpHandler) RegisterRoutes(r *chi.Mux) {
	r.Post("/tester/run", h.testerRun)
	r.Get("/tester/run/{evalUuid}", h.testerListen)
	r.Get("/exec/{execUuid}", h.execGet)
}
