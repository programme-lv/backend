package http

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/task/srvc"
)

type PutStatementReq struct {
	Story   string `json:"story"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Notes   string `json:"notes"`
	Scoring string `json:"scoring"`
	Talk    string `json:"talk"`
	Example string `json:"example"`
}

func (h *taskHttpHandler) PutStatement(ctx context.Context, req PutStatementReq) error {
	taskId := chi.URLParamFromCtx(ctx, "taskId")
	lang := chi.URLParamFromCtx(ctx, "langIso639")

	return h.taskSrvc.UpdateStatementMd(ctx, taskId, srvc.MarkdownStatement{
		LangIso639: lang,
		Story:      req.Story,
		Input:      req.Input,
		Output:     req.Output,
		Notes:      req.Notes,
		Scoring:    req.Scoring,
		Talk:       req.Talk,
		Example:    req.Example,
	})
}

func (h *taskHttpHandler) DeleteStatementImage(ctx context.Context) error {
	taskId := chi.URLParamFromCtx(ctx, "taskId")
	filename := chi.URLParamFromCtx(ctx, "filename")

	if err := h.taskSrvc.DeleteStatementImage(ctx, taskId, filename); err != nil {
		return err
	}

	h.cache.Delete(taskGetCacheKey(taskId))
	return nil
}
