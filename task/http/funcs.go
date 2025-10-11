package http

import (
	"context"
	"time"

	cache "github.com/Code-Hex/go-generics-cache"
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

	h.getTaskViewCache.Delete(taskId)
	return nil
}

func (h *taskHttpHandler) GetTaskView(ctx context.Context) (Task, error) {
	taskId := chi.URLParamFromCtx(ctx, "taskId")

	if h.getTaskViewCache.Contains(taskId) {
		task, _ := h.getTaskViewCache.Get(taskId)
		return task, nil
	}

	t, err := h.taskSrvc.GetTask(ctx, taskId)
	if err != nil {
		return Task{}, err
	}

	exp := cache.WithExpiration(5 * time.Second)
	response := h.mapTaskResponse(t)
	h.getTaskViewCache.Set(taskId, response, exp)
	return response, nil
}

func (h *taskHttpHandler) GetTaskList(ctx context.Context) ([]TaskPreview, error) {
	if h.getTaskListCache.Contains("") {
		previews, _ := h.getTaskListCache.Get("")
		return previews, nil
	}

	tasks, err := h.taskSrvc.ListTaskPreviews(ctx)
	if err != nil {
		return nil, err
	}
	previews := make([]TaskPreview, 0, len(tasks))
	for _, t := range tasks {
		previews = append(previews, h.mapTaskPreview(t))
	}
	exp := cache.WithExpiration(5 * time.Second)
	h.getTaskListCache.Set("", previews, exp)
	return previews, nil
}
