package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type PutStatementRequest struct {
	Story   string `json:"story"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Notes   string `json:"notes"`
	Scoring string `json:"scoring"`
	Talk    string `json:"talk"`
	Example string `json:"example"`
}

func (h *TaskHttpHandler) UpdateStatement(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	langIso639 := chi.URLParam(r, "langIso639")

	if langIso639 != "lv" {
		http.Error(w, "only lv is supported", http.StatusBadRequest)
		return
	}

	var req PutStatementRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("updateStatement", "taskId", taskId, "langIso639", langIso639, "req", req)

	// TODO: update statement

}
