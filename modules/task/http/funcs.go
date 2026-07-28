package http

import (
	"context"
	"time"

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

func (h *taskHttpHandler) GetTaskView(ctx context.Context) (Task, jsonresp.HttpStatusCoder) {
	taskId := chi.URLParamFromCtx(ctx, "taskId")

	if task, ok := h.getTaskViewCache.Get(taskId); ok {
		return task, nil
	}

	t, err := h.taskSrvc.GetTask(ctx, taskId)
	if err != nil {
		return Task{}, err
	}

	response := h.mapTaskResponse(t)
	if response.TaskFullName == "" {
		panic("task full name is empty")
	}

	h.getTaskViewCache.Set(taskId, response, 20*time.Second)
	return response, nil
}

func (h *taskHttpHandler) GetTaskList(ctx context.Context) ([]TaskPreview, jsonresp.HttpStatusCoder) {
	if previews, ok := h.getTaskListCache.Get(""); ok {
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

	h.getTaskListCache.Set("", previews, 20*time.Second)
	return previews, nil
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

// DeleteTaskOld deletes a task and all its related data (admin-only endpoint)
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

	// Clear any cached data for this task
	h.getTaskViewCache.Delete(taskId)
	h.getTaskListCache.Delete("")

	logger.Info("task deleted successfully", "task_id", taskId)

	return nil
}
