package storage

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Cleanup lists unreferenced public files. With apply, it moves them into a
// timestamped recovery directory outside public; it never destroys source bytes.
// Private tests and execution artifacts have separate retention requirements.
func (m Maintenance) Cleanup(ctx context.Context, apply bool) ([]string, error) {
	var pending int
	if err := m.DB.QueryRow(ctx, `SELECT count(*) FROM task_storage_migration`).Scan(&pending); err != nil {
		return nil, err
	}
	if pending > 0 {
		return nil, fmt.Errorf("%d storage migrations are still pending; resolve them before cleanup", pending)
	}
	live := make(map[string]bool)
	digests := make(map[string]string)
	sizes := make(map[string]int64)
	rows, err := m.DB.Query(ctx, `SELECT short_id, illustration_sha256, illustration_ext FROM tasks WHERE illustration_sha256 <> ''`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, digest, ext string
		if err := rows.Scan(&id, &digest, &ext); err != nil {
			rows.Close()
			return nil, err
		}
		live[Illustration(id, digest, ext)] = true
		digests[Illustration(id, digest, ext)] = digest
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows, err = m.DB.Query(ctx, `SELECT task_short_id, sha256, file_name FROM task_images`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, digest, name string
		if err := rows.Scan(&id, &digest, &name); err != nil {
			rows.Close()
			return nil, err
		}
		live[Image(id, digest, name)] = true
		digests[Image(id, digest, name)] = digest
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Keep derivatives only when the corresponding original is still referenced.
	var originals []string
	for key := range live {
		originals = append(originals, key)
	}
	for _, key := range originals {
		for _, variant := range []string{"list", "view", "full"} {
			live[key+"."+variant+".webp"] = false
		}
	}
	rows, err = m.DB.Query(ctx, `SELECT task_short_id, path, filesize_bytes FROM task_archive_files`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, name string
		var size int64
		if err := rows.Scan(&id, &name, &size); err != nil {
			rows.Close()
			return nil, err
		}
		live[Archive(id, name)] = true
		sizes[Archive(id, name)] = size
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	root := filepath.Join(m.Root, "public")
	for key, required := range live {
		if !required {
			continue
		}
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(key)))
		if err != nil {
			return nil, fmt.Errorf("referenced file %s: %w", key, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("referenced asset is not a regular file: %s", key)
		}
		if size, ok := sizes[key]; ok && info.Size() != size {
			return nil, fmt.Errorf("referenced asset size mismatch: %s", key)
		}
		if digest, ok := digests[key]; ok {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(key)))
			if err != nil {
				return nil, err
			}
			if Digest(data) != digest {
				return nil, fmt.Errorf("referenced asset checksum mismatch: %s", key)
			}
		}
	}
	var stale, dirs []string
	err = filepath.WalkDir(root, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, name)
			return nil
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		if _, ok := live[filepath.ToSlash(rel)]; !ok {
			stale = append(stale, rel)
		}
		return nil
	})
	if err != nil || !apply {
		return stale, err
	}
	recovery := filepath.Join(m.Root, "recovery", time.Now().UTC().Format("20060102T150405.000000000Z"), "public")
	for _, rel := range stale {
		src, dst := filepath.Join(root, rel), filepath.Join(recovery, rel)
		if err := mkdirDurable(filepath.Dir(dst)); err != nil {
			return stale, err
		}
		if err := os.Rename(src, dst); err != nil {
			return stale, err
		}
		if err := syncDir(filepath.Dir(dst)); err != nil {
			return stale, err
		}
		if err := syncDir(filepath.Dir(src)); err != nil {
			return stale, err
		}
	}
	for i := len(dirs) - 1; i > 0; i-- {
		entries, err := os.ReadDir(dirs[i])
		if err != nil {
			return stale, err
		}
		if len(entries) == 0 {
			if err := os.Remove(dirs[i]); err != nil {
				return stale, err
			}
		}
	}
	return stale, nil
}
