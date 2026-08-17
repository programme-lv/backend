package repo

import (
	"context"
	"fmt"

	"github.com/programme-lv/backend/modules/task/srvc"
)

func (r *taskPgRepo) AddStatementImg(ctx context.Context, taskId string, img srvc.StatementImage) error {
	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Defer rollback in case of error - transaction is committed later if successful
	defer tx.Rollback(ctx)

	// Check if the task exists
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id = $1)
	`, taskId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check if task exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("task with ID %s does not exist", taskId)
	}

	// Check if the image already exists for this task
	var imageExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM task_images WHERE task_short_id = $1 AND object_key = $2)
	`, taskId, img.ObjectKey).Scan(&imageExists)
	if err != nil {
		return fmt.Errorf("check if image exists: %w", err)
	}

	if imageExists {
		// Update existing image
		_, err = tx.Exec(ctx, `
			UPDATE task_images
			SET file_name = $3, width_px = $4, height_px = $5, filesize_bytes = $6
			WHERE task_short_id = $1 AND object_key = $2
		`, taskId, img.ObjectKey, img.Filename, img.WidthPx, img.HeightPx, img.SzInBytes)
		if err != nil {
			return fmt.Errorf("update statement image: %w", err)
		}
	} else {
		// Insert new image
		_, err = tx.Exec(ctx, `
			INSERT INTO task_images (task_short_id, object_key, file_name, width_px, height_px, filesize_bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, taskId, img.ObjectKey, img.Filename, img.WidthPx, img.HeightPx, img.SzInBytes)
		if err != nil {
			return fmt.Errorf("insert statement image: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// UpdateStatement implements srvc.TaskPgRepo.
func (r *taskPgRepo) UpdateStatement(ctx context.Context, taskId string, statement srvc.MarkdownStatement) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Defer rollback in case of error - transaction is committed later if successful
	defer tx.Rollback(ctx)

	// Check if the task exists
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id = $1)
	`, taskId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check if task exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("task with ID %s does not exist", taskId)
	}

	// Check if statement with this language already exists
	var statementExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM task_md_statements WHERE task_short_id = $1 AND lang_iso639 = $2)
	`, taskId, statement.LangIso639).Scan(&statementExists)
	if err != nil {
		return fmt.Errorf("check if statement exists: %w", err)
	}

	if statementExists {
		// Update existing statement
		_, err = tx.Exec(ctx, `
			UPDATE task_md_statements 
			SET story = $3, input = $4, output = $5, notes = $6, scoring = $7, talk = $8, example = $9
			WHERE task_short_id = $1 AND lang_iso639 = $2
		`, taskId, statement.LangIso639, statement.Story, statement.Input, statement.Output,
			statement.Notes, statement.Scoring, statement.Talk, statement.Example)
		if err != nil {
			return fmt.Errorf("update markdown statement: %w", err)
		}
	} else {
		// Insert new statement
		_, err = tx.Exec(ctx, `
			INSERT INTO task_md_statements (task_short_id, lang_iso639, story, input, output, notes, scoring, talk, example)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, taskId, statement.LangIso639, statement.Story, statement.Input, statement.Output,
			statement.Notes, statement.Scoring, statement.Talk, statement.Example)
		if err != nil {
			return fmt.Errorf("insert new markdown statement: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// DeleteStatementImg implements srvc.TaskPgRepo.
// It deletes an image from the database.
func (r *taskPgRepo) DeleteStatementImg(ctx context.Context, taskId string, filename string) error {
	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Defer rollback in case of error - transaction is committed later if successful
	defer tx.Rollback(ctx)

	// Check if the image exists
	var imageExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM task_images WHERE task_short_id = $1 AND file_name = $2)
	`, taskId, filename).Scan(&imageExists)
	if err != nil {
		return fmt.Errorf("check if image exists: %w", err)
	}
	if !imageExists {
		return fmt.Errorf("image with filename %s does not exist for task %s", filename, taskId)
	}

	// Delete the image
	_, err = tx.Exec(ctx, `
		DELETE FROM task_images
		WHERE task_short_id = $1 AND file_name = $2
	`, taskId, filename)
	if err != nil {
		return fmt.Errorf("delete statement image: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// UpdateIllustrationImg implements srvc.TaskPgRepo.
// It updates the illustration image fields in the tasks table.
func (r *taskPgRepo) UpdateIllustrationImg(ctx context.Context, taskId string, img srvc.IllustrationImage) error {
	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Defer rollback in case of error - transaction is committed later if successful
	defer tx.Rollback(ctx)

	// Check if the task exists
	var taskExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id = $1)
	`, taskId).Scan(&taskExists)
	if err != nil {
		return fmt.Errorf("check if task exists: %w", err)
	}
	if !taskExists {
		return fmt.Errorf("task with ID %s does not exist", taskId)
	}

	// Update the illustration image fields
	_, err = tx.Exec(ctx, `
		UPDATE tasks 
		SET illustr_img_object_key = $2, width_px = $3, height_px = $4, filesize_bytes = $5
		WHERE short_id = $1
	`, taskId, img.ObjectKey, img.WidthPx, img.HeightPx, img.SzInBytes)
	if err != nil {
		return fmt.Errorf("update illustration image: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
