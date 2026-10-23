package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/programme-lv/backend/modules/task/srvc"
)

func (r *taskPgRepo) UpdateTaskLimits(
	ctx context.Context, taskID string, cpuSecs float64, memMiB int,
) (int, error) {
	var revision int
	err := r.pool.QueryRow(ctx, `
		UPDATE tasks
		SET cpu_time_lim_secs = $2, mem_lim_megabytes = $3, revision = revision + 1
		WHERE short_id = $1
		RETURNING revision
	`, taskID, cpuSecs, memMiB).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, srvc.ErrTaskNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("update task limits: %w", err)
	}
	return revision, nil
}

func (r *taskPgRepo) UpdateTaskSolution(
	ctx context.Context, taskID, fname, content string, subtasks []int,
) (int, error) {
	subtasksJSON, err := json.Marshal(subtasks)
	if err != nil {
		return 0, fmt.Errorf("marshal solution subtasks: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin update solution: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `
		UPDATE task_solutions
		SET content = $3, subtasks = $4::jsonb
		WHERE task_id = $1 AND fname = $2
	`, taskID, fname, content, subtasksJSON)
	if err != nil {
		return 0, fmt.Errorf("update task solution: %w", err)
	}
	if result.RowsAffected() == 0 {
		return 0, srvc.ErrTaskSolutionNotFound
	}

	var revision int
	err = tx.QueryRow(ctx, `
		UPDATE tasks
		SET revision = revision + 1
		WHERE short_id = $1
		RETURNING revision
	`, taskID).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, srvc.ErrTaskNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("bump task revision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit solution update: %w", err)
	}
	return revision, nil
}
