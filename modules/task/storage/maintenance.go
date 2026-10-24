package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/filestore"
)

// Maintenance operates only while the caller owns Lock. The server invokes
// Prepare before starting workers or accepting requests.
type Maintenance struct {
	DB   *pgxpool.Pool
	Root string
}

// Prepare resumes asset conversion and any interrupted rename. A missing or
// conflicting file aborts startup rather than publishing a partial migration.
func (m Maintenance) Prepare(ctx context.Context) error {
	if err := m.bind(ctx); err != nil {
		return err
	}
	if err := m.migrate(ctx); err != nil {
		return err
	}
	if err := m.migrateLayout(ctx); err != nil {
		return err
	}
	if err := m.Recover(ctx); err != nil {
		return err
	}
	_, err := m.DB.Exec(ctx, `UPDATE task_storage_identity SET attached=true`)
	return err
}

// bind prevents maintenance against the right database but the wrong directory.
func (m Maintenance) bind(ctx context.Context) error {
	var token string
	var attached bool
	if err := m.DB.QueryRow(ctx, `SELECT token::text,attached FROM task_storage_identity`).Scan(&token, &attached); err != nil {
		return err
	}
	marker := filepath.Join(m.Root, ".task-storage-id")
	data, err := os.ReadFile(marker)
	if errors.Is(err, fs.ErrNotExist) && !attached {
		return copyVerified(marker, []byte(token+"\n"))
	}
	if err != nil {
		return fmt.Errorf("storage identity marker: %w", err)
	}
	if strings.TrimSpace(string(data)) != token {
		return fmt.Errorf("filestore belongs to a different database; use the matching database and storage root")
	}
	return nil
}

// LockDatabase prevents a second process from using the same DB with a different
// mount path. Close the returned session (do not return it to a live pool).
func LockDatabase(ctx context.Context, pool *pgxpool.Pool) (func(), error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	close := func() { conn.Conn().Close(context.Background()); conn.Release() }
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(345246485590)`).Scan(&locked); err != nil {
		close()
		return nil, err
	}
	if !locked {
		close()
		return nil, fmt.Errorf("database is in use; stop the backend before maintenance")
	}
	return close, nil
}

func (m Maintenance) migrate(ctx context.Context) error {
	store, err := filestore.NewStore(m.Root)
	if err != nil {
		return err
	}
	rows, err := m.DB.Query(ctx, `SELECT task_id, kind, filename, old_key FROM task_storage_migration ORDER BY task_id, kind, filename`)
	if err != nil {
		return err
	}
	type entry struct{ id, kind, filename, key string }
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.kind, &e.filename, &e.key); err != nil {
			rows.Close()
			return err
		}
		entries = append(entries, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, e := range entries {
		if e.kind == "zip" {
			// Leftover original zips need the task service's archive unpacking;
			// the server and CLI run that after Prepare.
			continue
		}
		if !ValidID(e.id) {
			return fmt.Errorf("invalid legacy task ID %q", e.id)
		}
		if e.kind != "illustration" && (!fs.ValidPath(e.filename) || e.filename == "." || strings.Contains(e.filename, "\\")) {
			return fmt.Errorf("invalid legacy asset filename %q", e.filename)
		}
		key := e.key
		switch e.kind {
		case "illustration":
			if !strings.Contains(key, "/") {
				key = "illustrations/" + key
			}
		case "image":
			if !strings.HasPrefix(key, "md-images/") && !strings.HasPrefix(key, "task/") && !strings.HasPrefix(key, "task-md-images/") {
				key = "md-images/" + key
			}
		}
		// 077 sources are the legacy public tree; destinations use the new layout.
		data, err := store.Download(path.Join("public", key))
		if err != nil {
			return fmt.Errorf("migrate %s %s: %w", e.id, key, err)
		}
		digest, ext := Digest(data), strings.ToLower(path.Ext(key))
		var dest string
		switch e.kind {
		case "illustration":
			dest = Illustration(e.id, digest, ext)
		case "image":
			dest = Image(e.id, digest, e.filename)
		case "archive":
			dest = Archive(e.id, e.filename)
		default:
			return fmt.Errorf("unknown asset kind %q", e.kind)
		}
		filename, err := store.Path(dest)
		if err != nil {
			return err
		}
		if err := copyVerified(filename, data); err != nil {
			return fmt.Errorf("copy %s: %w", dest, err)
		}
		tx, err := m.DB.Begin(ctx)
		if err != nil {
			return err
		}
		switch e.kind {
		case "illustration":
			_, err = tx.Exec(ctx, `UPDATE tasks SET illustration_sha256=$2, illustration_ext=$3 WHERE short_id=$1`, e.id, digest, ext)
		case "image":
			_, err = tx.Exec(ctx, `UPDATE task_images SET sha256=$3 WHERE task_short_id=$1 AND file_name=$2`, e.id, e.filename, digest)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM task_storage_migration WHERE task_id=$1 AND kind=$2 AND filename=$3`, e.id, e.kind, e.filename)
		}
		if err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// migrateLayout moves 077's public/tasks tree into the assets/archive/cache
// layout. Cache derivatives are deleted and regenerated lazily.
func (m Maintenance) migrateLayout(ctx context.Context) error {
	var layout string
	err := m.DB.QueryRow(ctx, `SELECT layout FROM task_storage_layout_pending`).Scan(&layout)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if layout != "assets-v1" {
		return fmt.Errorf("unknown storage layout migration %q", layout)
	}
	srcRoot := filepath.Join(m.Root, "public", "tasks")
	entries, readErr := os.ReadDir(srcRoot)
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		return readErr
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if !ValidID(id) {
			return fmt.Errorf("invalid task directory %q", id)
		}
		src := filepath.Join(srcRoot, id)
		if err := moveDir(filepath.Join(src, "images"), filepath.Join(m.Root, AssetsRoot, "tasks", id, "images")); err != nil {
			return err
		}
		if err := moveDir(filepath.Join(src, "illustration"), filepath.Join(m.Root, AssetsRoot, "tasks", id, "illustration")); err != nil {
			return err
		}
		if err := moveDir(filepath.Join(src, "archive"), filepath.Join(m.Root, ArchiveRoot, "tasks", id)); err != nil {
			return err
		}
		if err := removeDerived(filepath.Join(m.Root, AssetsRoot, "tasks", id)); err != nil {
			return err
		}
		remaining, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		if len(remaining) != 0 {
			return fmt.Errorf("unexpected entries left in %s", src)
		}
		if err := os.Remove(src); err != nil {
			return err
		}
	}
	remaining, readErr := os.ReadDir(srcRoot)
	if readErr == nil {
		if len(remaining) != 0 {
			return fmt.Errorf("public/tasks still contains non-directory entries")
		}
		if err := os.Remove(srcRoot); err != nil {
			return err
		}
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return readErr
	}
	// Remove empty legacy parents; ignore non-empty public/ because cleanup owns it.
	_ = os.Remove(filepath.Join(m.Root, "public"))
	if _, err := m.DB.Exec(ctx, `DELETE FROM task_storage_layout_pending`); err != nil {
		return err
	}
	return nil
}

func moveTaskDir(oldDir, newDir string) error {
	_, oldErr := os.Lstat(oldDir)
	_, newErr := os.Lstat(newDir)
	switch {
	case oldErr == nil && errors.Is(newErr, fs.ErrNotExist):
		return os.Rename(oldDir, newDir)
	case errors.Is(oldErr, fs.ErrNotExist) && newErr == nil:
		return nil // moved by an earlier attempt
	case errors.Is(oldErr, fs.ErrNotExist) && errors.Is(newErr, fs.ErrNotExist):
		return nil // task has no files in this root
	default:
		return fmt.Errorf("expected at most one of %s and %s", oldDir, newDir)
	}
}

func moveDir(src, dst string) error {
	if _, err := os.Lstat(src); errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("layout destination already exists: %s", dst)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := mkdirDurable(filepath.Dir(dst)); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	if err := syncDir(filepath.Dir(dst)); err != nil {
		return err
	}
	return syncDir(filepath.Dir(src))
}

func removeDerived(root string) error {
	err := filepath.WalkDir(root, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		for _, suffix := range []string{".list.webp", ".view.webp", ".full.webp"} {
			if strings.HasSuffix(name, suffix) {
				return os.Remove(name)
			}
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func copyVerified(filename string, data []byte) error {
	existing, err := os.ReadFile(filename)
	if err == nil {
		if !bytes.Equal(existing, data) {
			return fmt.Errorf("destination contains different bytes")
		}
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := mkdirDurable(filepath.Dir(filename)); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(filename), ".migrate-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), filename); err != nil {
		return err
	}
	if err := syncDir(filepath.Dir(filename)); err != nil {
		return err
	}
	got, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, data) {
		return fmt.Errorf("destination verification mismatch")
	}
	return nil
}

func mkdirDurable(dir string) error {
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(dir)
	if err := mkdirDurable(parent); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	return syncDir(parent)
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// Rename records durable intent, moves the task directory, then commits all
// relational references and the permanent alias in one transaction.
func (m Maintenance) Rename(ctx context.Context, oldID, newID string) error {
	if !ValidID(oldID) || !ValidID(newID) || oldID == newID {
		return fmt.Errorf("expected two distinct valid task IDs")
	}
	if err := m.Prepare(ctx); err != nil {
		return err
	}
	var oldExists, reserved bool
	if err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id=$1), EXISTS(SELECT 1 FROM tasks WHERE short_id=$2 UNION ALL SELECT 1 FROM task_aliases WHERE alias=$2)`, oldID, newID).Scan(&oldExists, &reserved); err != nil {
		return err
	}
	if !oldExists {
		return fmt.Errorf("task %s not found (use its canonical ID)", oldID)
	}
	if reserved {
		return fmt.Errorf("task ID %s is already in use or reserved", newID)
	}
	for _, root := range []string{AssetsRoot, ArchiveRoot} {
		newDir := filepath.Join(m.Root, root, "tasks", newID)
		if _, err := os.Lstat(newDir); !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("destination directory is occupied or inaccessible: %s", newDir)
		}
	}
	if _, err := m.DB.Exec(ctx, `INSERT INTO task_rename_pending(old_id,new_id) VALUES($1,$2)`, oldID, newID); err != nil {
		return err
	}
	return m.Recover(ctx)
}

// Recover rolls forward a durable rename intent. It is also called on startup,
// before any worker can observe the intermediate filesystem state.
func (m Maintenance) Recover(ctx context.Context) error {
	var oldID, newID string
	err := m.DB.QueryRow(ctx, `SELECT old_id,new_id FROM task_rename_pending`).Scan(&oldID, &newID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !ValidID(oldID) || !ValidID(newID) {
		return fmt.Errorf("invalid pending rename")
	}
	for _, root := range []string{AssetsRoot, ArchiveRoot} {
		parent := filepath.Join(m.Root, root, "tasks")
		if err := moveTaskDir(filepath.Join(parent, oldID), filepath.Join(parent, newID)); err != nil {
			return fmt.Errorf("recover rename %s -> %s: %w", oldID, newID, err)
		}
		if err := syncDir(parent); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	date := time.Now().UTC().Format("2006-01-02")
	noteLV := fmt.Sprintf("\n\nUzdevuma ID normalizēts %s: `%s` → `%s`. Uzdevuma saturs un vērtēšana nav mainīti.\n", date, oldID, newID)
	noteEN := fmt.Sprintf("\n\nTask ID normalized on %s: `%s` → `%s`. Task content and judging unchanged.\n", date, oldID, newID)
	tag, err := tx.Exec(ctx, `UPDATE tasks SET short_id=$2, revision=revision+1, readme=readme || CASE WHEN orig_lang='lv' THEN $3 ELSE $4 END WHERE short_id=$1`, oldID, newID, noteLV, noteEN)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("pending rename source task is missing")
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET task_shortid=$2 WHERE task_shortid=$1`, oldID, newID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO task_aliases(alias,task_id) VALUES($1,$2)`, oldID, newID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM task_rename_pending`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
