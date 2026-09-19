package repo

import (
	"context"
	"fmt"

	"github.com/programme-lv/backend/modules/task/srvc"
)

func (r *taskPgRepo) ListArchiveFiles(ctx context.Context, taskId string) ([]srvc.ArchiveFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT path, object_key, filesize_bytes
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
		if err := rows.Scan(&file.Path, &file.ObjectKey, &file.SzInBytes); err != nil {
			return nil, fmt.Errorf("scan archive file: %w", err)
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (r *taskPgRepo) AddArchiveFile(ctx context.Context, taskId string, file srvc.ArchiveFile) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO task_archive_files (task_short_id, path, object_key, filesize_bytes)
		VALUES ($1, $2, $3, $4)
	`, taskId, file.Path, file.ObjectKey, file.SzInBytes)
	if err != nil {
		return fmt.Errorf("insert archive file: %w", err)
	}
	return nil
}

func (r *taskPgRepo) DeleteArchiveFile(ctx context.Context, taskId, relPath string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM task_archive_files
		WHERE task_short_id = $1 AND path = $2
	`, taskId, relPath)
	if err != nil {
		return fmt.Errorf("delete archive file: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("archive file %s not found", relPath)
	}
	return nil
}

func (r *taskPgRepo) ListLegacyArchiveZips(ctx context.Context) ([]srvc.LegacyArchiveZip, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT short_id, archive_object_key
		FROM tasks
		WHERE archive_object_key <> ''
		ORDER BY short_id
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
		UPDATE tasks SET archive_object_key = '' WHERE short_id = $1
	`, taskId)
	if err != nil {
		return fmt.Errorf("clear archive object key: %w", err)
	}
	return nil
}
