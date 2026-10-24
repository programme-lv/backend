package repo

import (
	"context"
	"fmt"

	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/programme-lv/backend/modules/task/storage"
)

func (r *taskPgRepo) ListArchiveFiles(ctx context.Context, taskId string) ([]srvc.ArchiveFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT path, filesize_bytes
		FROM task_archive_files
		WHERE task_short_id = $1
		ORDER BY path
	`, taskId)
	if err != nil {
		return nil, fmt.Errorf("list archive files: %w", err)
	}
	defer rows.Close()

	var files []srvc.ArchiveFile
	for rows.Next() {
		var file srvc.ArchiveFile
		if err := rows.Scan(&file.Path, &file.SzInBytes); err != nil {
			return nil, fmt.Errorf("scan archive file: %w", err)
		}
		file.ObjectKey = storage.Archive(taskId, file.Path)
		files = append(files, file)
	}
	return files, rows.Err()
}

func (r *taskPgRepo) AddArchiveFile(ctx context.Context, taskId string, file srvc.ArchiveFile) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO task_archive_files (task_short_id, path, filesize_bytes)
		VALUES ($1, $2, $3)
	`, taskId, file.Path, file.SzInBytes)
	if err != nil {
		return fmt.Errorf("insert archive file: %w", err)
	}
	if err := bumpRevision(ctx, tx, taskId); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *taskPgRepo) DeleteArchiveFile(ctx context.Context, taskId, relPath string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		DELETE FROM task_archive_files
		WHERE task_short_id = $1 AND path = $2
	`, taskId, relPath)
	if err != nil {
		return fmt.Errorf("delete archive file: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("archive file %s not found", relPath)
	}
	if err := bumpRevision(ctx, tx, taskId); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListLegacyArchiveZips returns leftover original archive zips waiting in the
// one-time storage migration queue.
func (r *taskPgRepo) ListLegacyArchiveZips(ctx context.Context) ([]srvc.LegacyArchiveZip, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT task_id, old_key FROM task_storage_migration WHERE kind = 'zip' ORDER BY task_id
	`)
	if err != nil {
		return nil, fmt.Errorf("list legacy archive zips: %w", err)
	}
	defer rows.Close()

	var zips []srvc.LegacyArchiveZip
	for rows.Next() {
		var z srvc.LegacyArchiveZip
		if err := rows.Scan(&z.TaskID, &z.ObjectKey); err != nil {
			return nil, fmt.Errorf("scan legacy archive zip: %w", err)
		}
		zips = append(zips, z)
	}
	return zips, rows.Err()
}

func (r *taskPgRepo) ClearArchiveObjectKey(ctx context.Context, taskId string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM task_storage_migration WHERE kind = 'zip' AND task_id = $1
	`, taskId)
	if err != nil {
		return fmt.Errorf("clear archive object key: %w", err)
	}
	return nil
}
