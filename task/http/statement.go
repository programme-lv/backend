package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/task/srvc"
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

func (h *TaskHttpHandler) PutStatement(w http.ResponseWriter, r *http.Request) {
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

	err = h.taskSrvc.UpdateStatement(r.Context(), taskId, srvc.MarkdownStatement{
		LangIso639: langIso639,
		Story:      req.Story,
		Input:      req.Input,
		Output:     req.Output,
		Notes:      req.Notes,
		Scoring:    req.Scoring,
		Talk:       req.Talk,
		Example:    req.Example,
	})
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	httpjson.WriteSuccessJson(w, nil)
}
