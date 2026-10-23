package http

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
)

type UpdateTaskLimitsReq struct {
	CPUMs  int `json:"cpu_ms"`
	MemMiB int `json:"mem_mib"`
}

type UpdateTaskLimitsResp struct {
	Revision int `json:"revision"`
}

type UpdateTaskSolutionReq struct {
	Content  string `json:"content"`
	Subtasks []int  `json:"subtasks"`
}

type UpdateTaskSolutionResp struct {
	Revision int `json:"revision"`
}

type TaskSolutionView struct {
	Fname    string `json:"fname"`
	Content  string `json:"content"`
	Subtasks []int  `json:"subtasks"`
}

func (h *taskHttpHandler) UpdateTaskLimits(ctx context.Context, req UpdateTaskLimitsReq) (UpdateTaskLimitsResp, jsonresp.HttpStatusCoder) {
	taskID := chi.URLParamFromCtx(ctx, "taskId")
	revision, err := h.taskSrvc.UpdateTaskLimits(ctx, taskID, req.CPUMs, req.MemMiB)
	if err != nil {
		return UpdateTaskLimitsResp{}, err
	}
	h.invalidateTaskCaches(taskID)
	return UpdateTaskLimitsResp{Revision: revision}, nil
}

func (h *taskHttpHandler) UpdateTaskSolution(ctx context.Context, req UpdateTaskSolutionReq) (UpdateTaskSolutionResp, jsonresp.HttpStatusCoder) {
	taskID := chi.URLParamFromCtx(ctx, "taskId")
	fname := chi.URLParamFromCtx(ctx, "fname")
	revision, err := h.taskSrvc.UpdateTaskSolution(ctx, taskID, fname, req.Content, req.Subtasks)
	if err != nil {
		return UpdateTaskSolutionResp{}, err
	}
	h.invalidateTaskCaches(taskID)
	return UpdateTaskSolutionResp{Revision: revision}, nil
}

func (h *taskHttpHandler) GetTaskSolution(ctx context.Context) (TaskSolutionView, jsonresp.HttpStatusCoder) {
	taskID := chi.URLParamFromCtx(ctx, "taskId")
	fname := chi.URLParamFromCtx(ctx, "fname")
	solution, err := h.taskSrvc.GetTaskSolution(ctx, taskID, fname)
	if err != nil {
		return TaskSolutionView{}, err
	}
	return TaskSolutionView{
		Fname:    solution.Fname,
		Content:  solution.Content,
		Subtasks: solution.Subtasks,
	}, nil
}

func (h *taskHttpHandler) invalidateTaskCaches(taskID string) {
	h.getTaskViewCache.Delete(taskID)
	h.getTaskListCache.Delete("")
	h.getTaskFiltersCache.Delete("")
}
