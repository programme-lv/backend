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

func (h *taskHttpHandler) PutStatement(ctx context.Context, req PutStatementReq) (Empty, error) {
	taskId := chi.URLParamFromCtx(ctx, "taskId")
	lang := chi.URLParamFromCtx(ctx, "langIso639")

	err := h.taskSrvc.UpdateStatementMd(ctx, taskId, srvc.MarkdownStatement{
		LangIso639: lang,
		Story:      req.Story,
		Input:      req.Input,
		Output:     req.Output,
		Notes:      req.Notes,
		Scoring:    req.Scoring,
		Talk:       req.Talk,
		Example:    req.Example,
	})
	return Empty{}, err
}

func (h *taskHttpHandler) DeleteStatementImage(ctx context.Context, req Empty) (Empty, error) {
	taskId := chi.URLParamFromCtx(ctx, "taskId")
	filename := chi.URLParamFromCtx(ctx, "filename")

	err := h.taskSrvc.DeleteStatementImage(ctx, taskId, filename)
	if err != nil {
		return Empty{}, err
	}

	h.cache.Delete(taskGetCacheKey(taskId))

	return Empty{}, nil
}
