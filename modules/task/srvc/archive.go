package srvc

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"path"
	"strings"

	"github.com/programme-lv/backend/common/srvcerror"
	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
)

const (
	taskArchiveDir       = "archive"
	maxArchiveFiles      = 256
	maxArchiveFileBytes  = 64 << 20
	maxArchiveTotalBytes = 128 << 20
	legacyArchiveSkipDir = "tests"
)

func archiveUsage(files []ArchiveFile) (n int, total int) {
	for _, file := range files {
		n++
		total += file.SzInBytes
	}
	return n, total
}

func archiveAddsExceedQuota(existing []ArchiveFile, addCount, addBytes int) srvcerror.E {
	n, total := archiveUsage(existing)
	if n+addCount > maxArchiveFiles {
		return ErrArchiveTooManyFiles
	}
	if total+addBytes > maxArchiveTotalBytes {
		return ErrArchiveTotalTooLarge
	}
	return nil
}

func taskArchiveObjectKey(taskID, relPath string) string {
	return path.Join(taskArchiveDir, taskID, relPath)
}

func cleanArchiveRelPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, `\`, "/")
	raw = strings.TrimPrefix(raw, "/")
	raw = strings.TrimPrefix(raw, "archive/")
	if raw == "" {
		return "", fmt.Errorf("empty archive path")
	}
	clean, err := taskzipv1.SafeRelPath(raw)
	if err != nil {
		return "", err
	}
	return clean, nil
}

func (ts *taskSrvc) ListArchiveFiles(ctx context.Context, taskId string) ([]ArchiveFile, srvcerror.E) {
	exists, err := ts.repo.Exists(ctx, taskId)
	if err != nil {
		ts.logger(ctx).Error("check task exists", "error", err)
		return nil, srvcerror.InternalServerError()
	}
	if !exists {
		return nil, errTaskNotFound(taskId)
	}
	files, err := ts.repo.ListArchiveFiles(ctx, taskId)
	if err != nil {
		ts.logger(ctx).Error("list archive files", "error", err)
		return nil, srvcerror.InternalServerError()
	}
	if files == nil {
		files = []ArchiveFile{}
	}
	return files, nil
}

func (ts *taskSrvc) DownloadArchiveFile(ctx context.Context, taskId, relPath string) ([]byte, string, srvcerror.E) {
	relPath, err := cleanArchiveRelPath(relPath)
	if err != nil {
		return nil, "", ErrInvalidArchivePath
	}
	files, listErr := ts.ListArchiveFiles(ctx, taskId)
	if listErr != nil {
		return nil, "", listErr
	}
	var found *ArchiveFile
	for i := range files {
		if files[i].Path == relPath {
			found = &files[i]
			break
		}
	}
	if found == nil {
		return nil, "", errArchiveFileNotFound(relPath)
	}
	data, err := ts.publicStore.Download(found.ObjectKey)
	if err != nil {
		ts.logger(ctx).Error("download archive file", "error", err, "path", relPath)
		return nil, "", srvcerror.InternalServerError()
	}
	return data, path.Base(relPath), nil
}

func (ts *taskSrvc) UploadArchiveFile(ctx context.Context, taskId, relPath string, body []byte) srvcerror.E {
	if len(body) > maxArchiveFileBytes {
		return ErrArchiveFileTooLarge
	}
	relPath, err := cleanArchiveRelPath(relPath)
	if err != nil {
		return ErrInvalidArchivePath
	}
	exists, err := ts.repo.Exists(ctx, taskId)
	if err != nil {
		ts.logger(ctx).Error("check task exists", "error", err)
		return srvcerror.InternalServerError()
	}
	if !exists {
		return errTaskNotFound(taskId)
	}
	files, err := ts.repo.ListArchiveFiles(ctx, taskId)
	if err != nil {
		ts.logger(ctx).Error("list archive files", "error", err)
		return srvcerror.InternalServerError()
	}
	for _, file := range files {
		if file.Path == relPath {
			return errArchiveFileAlreadyExists(relPath)
		}
	}
	if err := archiveAddsExceedQuota(files, 1, len(body)); err != nil {
		return err
	}
	return ts.storeArchiveFile(ctx, taskId, relPath, body)
}

func (ts *taskSrvc) storeArchiveFile(ctx context.Context, taskId, relPath string, body []byte) srvcerror.E {
	objectKey := taskArchiveObjectKey(taskId, relPath)
	mediaType := mime.TypeByExtension(path.Ext(relPath))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	if _, err := ts.publicStore.Upload(body, objectKey, mediaType); err != nil {
		ts.logger(ctx).Error("upload archive file", "error", err, "path", relPath)
		return srvcerror.InternalServerError()
	}
	if err := ts.repo.AddArchiveFile(ctx, taskId, ArchiveFile{
		Path: relPath, ObjectKey: objectKey, SzInBytes: len(body),
	}); err != nil {
		ts.logger(ctx).Error("insert archive file", "error", err, "path", relPath)
		return srvcerror.InternalServerError()
	}
	return nil
}

func (ts *taskSrvc) DeleteArchiveFile(ctx context.Context, taskId, relPath string) srvcerror.E {
	relPath, err := cleanArchiveRelPath(relPath)
	if err != nil {
		return ErrInvalidArchivePath
	}
	exists, err := ts.repo.Exists(ctx, taskId)
	if err != nil {
		ts.logger(ctx).Error("check task exists", "error", err)
		return srvcerror.InternalServerError()
	}
	if !exists {
		return errTaskNotFound(taskId)
	}
	files, err := ts.repo.ListArchiveFiles(ctx, taskId)
	if err != nil {
		ts.logger(ctx).Error("list archive files", "error", err)
		return srvcerror.InternalServerError()
	}
	var objectKey string
	for _, file := range files {
		if file.Path == relPath {
			objectKey = file.ObjectKey
			break
		}
	}
	if objectKey == "" {
		return errArchiveFileNotFound(relPath)
	}
	if err := ts.repo.DeleteArchiveFile(ctx, taskId, relPath); err != nil {
		ts.logger(ctx).Error("delete archive file row", "error", err, "path", relPath)
		return srvcerror.InternalServerError()
	}
	if err := ts.publicStore.Delete(objectKey); err != nil {
		ts.logger(ctx).Error("delete archive file object", "error", err, "path", relPath)
	}
	return nil
}

func (ts *taskSrvc) uploadTaskZipArchive(ctx context.Context, task *Task, files map[string][]byte) srvcerror.E {
	addBytes := 0
	for rel, data := range files {
		if _, err := cleanArchiveRelPath(rel); err != nil {
			return errInvalidTaskZip(err.Error())
		}
		if len(data) > maxArchiveFileBytes {
			return ErrArchiveFileTooLarge
		}
		addBytes += len(data)
	}
	if err := archiveAddsExceedQuota(nil, len(files), addBytes); err != nil {
		return err
	}
	for rel, data := range files {
		rel, err := cleanArchiveRelPath(rel)
		if err != nil {
			return errInvalidTaskZip(err.Error())
		}
		objectKey := taskArchiveObjectKey(task.ShortId, rel)
		mediaType := mime.TypeByExtension(path.Ext(rel))
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		if _, err := ts.publicStore.Upload(data, objectKey, mediaType); err != nil {
			ts.logger(ctx).Error("upload imported archive file", "error", err, "path", rel)
			return srvcerror.InternalServerError()
		}
		task.Archive = append(task.Archive, ArchiveFile{
			Path: rel, ObjectKey: objectKey, SzInBytes: len(data),
		})
	}
	return nil
}

func (ts *taskSrvc) downloadTaskZipArchive(ctx context.Context, task Task, dest map[string][]byte) error {
	for _, file := range task.Archive {
		data, err := ts.publicStore.Download(file.ObjectKey)
		if err != nil {
			return fmt.Errorf("download archive %s: %w", file.Path, err)
		}
		dest[file.Path] = data
	}
	return nil
}

func (ts *taskSrvc) exportIllustrationToArchive(ctx context.Context, task Task, dest map[string][]byte) error {
	if task.IllustrImg == nil || task.IllustrImg.ObjectKey == "" {
		return nil
	}
	ext := path.Ext(task.IllustrImg.ObjectKey)
	if ext == "" {
		ext = ".png"
	}
	rel := "illustration" + ext
	if _, exists := dest[rel]; exists {
		return nil
	}
	data, err := ts.publicStore.Download(taskIllustrationObjectKey(task.IllustrImg.ObjectKey))
	if err != nil {
		return fmt.Errorf("download illustration: %w", err)
	}
	dest[rel] = data
	return nil
}

// MigrateLegacyArchiveZips unpacks leftover tasks.archive_object_key zips
// into task_archive_files. Missing zips are left for a later start.
func (ts *taskSrvc) MigrateLegacyArchiveZips(ctx context.Context) {
	zips, err := ts.repo.ListLegacyArchiveZips(ctx)
	if err != nil {
		ts.logger(ctx).Error("list legacy archive zips", "error", err)
		return
	}
	if len(zips) == 0 {
		return
	}
	ts.logger(ctx).Info("migrating leftover task archive zips", "count", len(zips))
	for _, z := range zips {
		if err := ts.migrateOneLegacyArchiveZip(ctx, z); err != nil {
			ts.logger(ctx).Error("migrate leftover archive zip", "task_id", z.TaskID, "error", err)
		}
	}
}

func (ts *taskSrvc) migrateOneLegacyArchiveZip(ctx context.Context, z LegacyArchiveZip) error {
	exists, err := ts.publicStore.Exists(z.ObjectKey)
	if err != nil {
		return err
	}
	if !exists {
		ts.logger(ctx).Warn("legacy archive zip missing", "task_id", z.TaskID, "object_key", z.ObjectKey)
		return nil
	}
	data, err := ts.publicStore.Download(z.ObjectKey)
	if err != nil {
		return err
	}
	files, err := extractLegacyArchiveZip(data)
	if err != nil {
		return err
	}
	existing, err := ts.repo.ListArchiveFiles(ctx, z.TaskID)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, file := range existing {
		have[file.Path] = true
	}
	addCount, addBytes := 0, 0
	for rel, body := range files {
		if have[rel] {
			continue
		}
		if len(body) > maxArchiveFileBytes {
			return ErrArchiveFileTooLarge
		}
		addCount++
		addBytes += len(body)
	}
	if err := archiveAddsExceedQuota(existing, addCount, addBytes); err != nil {
		return err
	}
	for rel, body := range files {
		if have[rel] {
			continue
		}
		if err := ts.storeArchiveFile(ctx, z.TaskID, rel, body); err != nil {
			return fmt.Errorf("store %s: %w", rel, err)
		}
	}
	if err := ts.repo.ClearArchiveObjectKey(ctx, z.TaskID); err != nil {
		return err
	}
	if err := ts.publicStore.Delete(z.ObjectKey); err != nil {
		ts.logger(ctx).Error("delete leftover archive zip", "task_id", z.TaskID, "error", err)
	}
	return nil
}

func extractLegacyArchiveZip(data []byte) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}
	raw := map[string][]byte{}
	hasTaskTOML := false
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name, err := taskzipv1.SafeRelPath(strings.TrimSuffix(f.Name, "/"))
		if err != nil {
			continue
		}
		if path.Base(name) == "task.toml" {
			hasTaskTOML = true
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(rc, maxArchiveFileBytes+1))
		rc.Close()
		if err != nil {
			return nil, err
		}
		if len(body) > maxArchiveFileBytes {
			return nil, fmt.Errorf("archive file %s exceeds %d bytes", name, maxArchiveFileBytes)
		}
		raw[name] = body
	}
	out := map[string][]byte{}
	if hasTaskTOML {
		for name, body := range raw {
			if !strings.HasPrefix(name, "archive/") {
				continue
			}
			rel := strings.TrimPrefix(name, "archive/")
			if rel != "" {
				out[rel] = body
			}
		}
		return out, nil
	}
	for name, body := range raw {
		if skipLegacyArchivePath(name) {
			continue
		}
		out[name] = body
	}
	return out, nil
}

func skipLegacyArchivePath(name string) bool {
	first, _, _ := strings.Cut(name, "/")
	if first == legacyArchiveSkipDir || first == "testspec" || first == "__MACOSX" || first == ".git" {
		return true
	}
	return path.Base(name) == ".DS_Store"
}
