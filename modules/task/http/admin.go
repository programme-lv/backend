package http

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/task/srvc"
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

func (h *taskHttpHandler) PutStatement(ctx context.Context, req PutStatementReq) jsonresp.HttpStatusCoder {
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
	if err != nil {
		return err
	}
	return nil
}

func (h *taskHttpHandler) DeleteStatementImage(ctx context.Context) jsonresp.HttpStatusCoder {
	taskId := chi.URLParamFromCtx(ctx, "taskId")
	filename := chi.URLParamFromCtx(ctx, "filename")

	err := h.taskSrvc.DeleteStatementImage(ctx, taskId, filename)
	if err != nil {
		return err
	}

	h.getTaskViewCache.Delete(taskId)
	h.getTaskListCache.Delete("")
	return nil
}

func (h *taskHttpHandler) DeleteIllustration(ctx context.Context) jsonresp.HttpStatusCoder {
	taskId := chi.URLParamFromCtx(ctx, "taskId")

	err := h.taskSrvc.DeleteIllustrationImg(ctx, taskId)
	if err != nil {
		return err
	}

	h.getTaskViewCache.Delete(taskId)
	h.getTaskListCache.Delete("")

	return nil
}

func (h *taskHttpHandler) DeleteTask(ctx context.Context) jsonresp.HttpStatusCoder {
	logger := h.logger(ctx)
	taskId := chi.URLParamFromCtx(ctx, "taskId")

	if taskId == "" {
		return jsonresp.ErrHttpBadRequest.WithMsg("task ID is required")
	}

	err := h.taskSrvc.DeleteTask(ctx, taskId)
	if err != nil {
		return err
	}

	h.getTaskViewCache.Delete(taskId)
	h.getTaskListCache.Delete("")

	logger.Info("task deleted successfully", "task_id", taskId)

	return nil
}
