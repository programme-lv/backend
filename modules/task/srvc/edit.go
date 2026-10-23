package srvc

import (
	"context"
	"errors"

	"github.com/programme-lv/backend/common/srvcerror"
)

const MaxSolutionLength = 1_000_000

// UpdateTaskLimits changes only the execution limits. It intentionally does
// not bump tests_version: existing profile measurements remain valid and the
// calibration is recomputed at the new limits.
func (ts *taskSrvc) UpdateTaskLimits(ctx context.Context, taskID string, cpuMs, memMiB int) (int, srvcerror.E) {
	if cpuMs < 1 || cpuMs > 10_000 {
		return 0, ErrInvalidTaskLimits.WithMsg("CPU laika limits jābūt no 1 ms līdz 10 s")
	}
	if memMiB < 1 || memMiB > 1024 {
		return 0, ErrInvalidTaskLimits.WithMsg("Atmiņas limits jābūt no 1 MB līdz 1024 MB")
	}

	revision, err := ts.repo.UpdateTaskLimits(ctx, taskID, float64(cpuMs)/1000, memMiB)
	if err != nil {
		var serviceErr srvcerror.E
		if errors.As(err, &serviceErr) {
			return 0, serviceErr
		}
		ts.logger(ctx).Error("update task limits", "task_id", taskID, "error", err)
		return 0, srvcerror.InternalServerError()
	}
	return revision, nil
}

// UpdateTaskSolution changes a reference solution's source and expected
// subtasks. It does not bump tests_version; the solution hash makes the old
// profile visibly stale instead.
func (ts *taskSrvc) UpdateTaskSolution(
	ctx context.Context, taskID, fname, content string, subtasks []int,
) (int, srvcerror.E) {
	if content == "" {
		return 0, ErrInvalidTaskSolution.WithMsg("Risinājuma kods nedrīkst būt tukšs")
	}
	if len(content) > MaxSolutionLength {
		return 0, ErrInvalidTaskSolution.WithMsg("Risinājuma kods ir pārāk garš")
	}

	task, err := ts.GetTask(ctx, taskID)
	if err != nil {
		return 0, err
	}
	if _, ok := findTaskSolution(task, fname); !ok {
		return 0, ErrTaskSolutionNotFound
	}

	seen := make(map[int]bool, len(subtasks))
	unique := make([]int, 0, len(subtasks))
	for _, id := range subtasks {
		if id < 1 || id > len(task.Subtasks) {
			return 0, ErrInvalidTaskSolution.WithMsg("Apakšuzdevuma ID nav derīgs")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}

	revision, repoErr := ts.repo.UpdateTaskSolution(ctx, taskID, fname, content, unique)
	if repoErr != nil {
		var serviceErr srvcerror.E
		if errors.As(repoErr, &serviceErr) {
			return 0, serviceErr
		}
		ts.logger(ctx).Error("update task solution", "task_id", taskID, "fname", fname, "error", repoErr)
		return 0, srvcerror.InternalServerError()
	}
	return revision, nil
}

func (ts *taskSrvc) GetTaskSolution(ctx context.Context, taskID, fname string) (Solution, srvcerror.E) {
	task, err := ts.GetTask(ctx, taskID)
	if err != nil {
		return Solution{}, err
	}
	solution, ok := findTaskSolution(task, fname)
	if !ok {
		return Solution{}, ErrTaskSolutionNotFound
	}
	return solution, nil
}

func findTaskSolution(task Task, fname string) (Solution, bool) {
	for _, solution := range task.Solutions {
		if solution.Fname == fname {
			return solution, true
		}
	}
	return Solution{}, false
}
