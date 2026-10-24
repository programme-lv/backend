//go:build integration

package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/testutil"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/programme-lv/backend/modules/subm/pgrepo"
	"github.com/programme-lv/backend/modules/task/repo"
	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/programme-lv/backend/modules/task/storage"
	taskzip "github.com/programme-lv/backend/modules/task/taskzip"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (storage.Maintenance, *pgxpool.Pool, *filestore.Store) {
	t.Helper()
	pool := testutil.MustGetMigratedTestPostgresDb(t)
	root := t.TempDir()
	lock, err := storage.Lock(root)
	require.NoError(t, err)
	t.Cleanup(func() { lock.Close() })
	store, err := filestore.NewStore(filepath.Join(root, "public"))
	require.NoError(t, err)
	return storage.Maintenance{DB: pool, Root: root}, pool, store
}

func seedTask(t *testing.T, pool *pgxpool.Pool, id string) srvc.Task {
	t.Helper()
	task := srvc.Task{ShortId: id, FullName: map[string]string{"lv": "Task"}, OrigLang: "lv", MemLimMegabytes: 64, CpuTimeLimSecs: 1, Readme: "Original notes", Solutions: []srvc.Solution{{Fname: "main.cpp", Content: "int main(){}"}}, Tests: []srvc.Test{{InpSha2: "input", AnsSha2: "answer"}}}
	require.NoError(t, repo.NewTaskPgRepo(pool).CreateTask(context.Background(), task))
	got, err := repo.NewTaskPgRepo(pool).GetTask(context.Background(), id)
	require.NoError(t, err)
	return got
}

func TestRenamePreservesHistoryAndAssets(t *testing.T) {
	m, pool, store := setup(t)
	ctx := context.Background()
	before := seedTask(t, pool, "seifs")
	data := []byte("original source bytes\r\n")
	_, err := store.Upload(data, storage.Archive("seifs", "source.txt"), "text/plain")
	require.NoError(t, err)
	require.NoError(t, repo.NewTaskPgRepo(pool).AddArchiveFile(ctx, "seifs", srvc.ArchiveFile{Path: "source.txt", SzInBytes: len(data)}))
	user, eval := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO users(uuid,firstname,lastname,username,email,bcrypt_pwd) VALUES($1,'Test','User','rename','rename@example.com','test')`, user)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO evaluations(uuid,stage,score_unit,cpu_lim_ms,mem_lim_kib) VALUES($1,'finished','test',1000,65536)`, eval)
	require.NoError(t, err)
	subm := domain.Subm{UUID: uuid.New(), Content: "original submission", AuthorUUID: user, TaskShortID: "seifs", LangShortID: "cpp17", CurrEvalUUID: eval, CreatedAt: time.Now()}
	subms := pgrepo.NewPgSubmRepo(pool)
	require.NoError(t, subms.StoreSubm(ctx, &subm))
	profileID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO task_solution_profiles(id,task_id,fname,lang_id,status,expected_points,possible_points,tests_version,solution_sha256) VALUES($1,'seifs','main.cpp','cpp17','finished',100,100,1,'sha')`, profileID)
	require.NoError(t, err)
	require.NoError(t, m.Rename(ctx, "seifs", "lio2025seifs"))
	after, err := repo.NewTaskPgRepo(pool).GetTask(ctx, "seifs")
	require.NoError(t, err)
	require.Equal(t, "lio2025seifs", after.ShortId)
	require.Equal(t, before.CreatedAt, after.CreatedAt)
	require.Equal(t, before.TestsVersion, after.TestsVersion)
	require.Equal(t, before.Tests, after.Tests)
	require.Equal(t, before.Solutions, after.Solutions)
	require.Contains(t, after.Readme, "Original notes")
	require.Contains(t, after.Readme, "`seifs` → `lio2025seifs`")
	got, err := store.Download(after.Archive[0].ObjectKey)
	require.NoError(t, err)
	require.Equal(t, data, got)
	require.NoDirExists(t, filepath.Join(m.Root, "public", "tasks", "seifs"))
	saved, err := subms.GetSubm(ctx, subm.UUID)
	require.NoError(t, err)
	require.Equal(t, subm.ShortID, saved.ShortID)
	require.Equal(t, subm.Content, saved.Content)
	require.Equal(t, eval, saved.CurrEvalUUID)
	require.Equal(t, after.ShortId, saved.TaskShortID)
	count, err := subms.CountSubms(ctx, nil, "seifs", nil, nil, nil, true)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	var profileTask string
	require.NoError(t, pool.QueryRow(ctx, `SELECT task_id FROM task_solution_profiles WHERE id=$1`, profileID).Scan(&profileTask))
	require.Equal(t, after.ShortId, profileTask)
	// A second rename flattens aliases and reserves all historical IDs.
	require.NoError(t, m.Rename(ctx, "lio2025seifs", "lio2025seifs-final"))
	for _, old := range []string{"seifs", "lio2025seifs"} {
		id, err := repo.NewTaskPgRepo(pool).ResolveID(ctx, old)
		require.NoError(t, err)
		require.Equal(t, "lio2025seifs-final", id)
		require.Error(t, repo.NewTaskPgRepo(pool).CreateTask(ctx, srvc.Task{ShortId: old}))
	}
	require.Error(t, m.Rename(ctx, "lio2025seifs-final", "seifs"))
	require.Error(t, m.Rename(ctx, "lio2025seifs-final", "../escape"))
	_, err = store.Upload([]byte("unrelated leftover"), storage.Archive("occupied", "x"), "text/plain")
	require.NoError(t, err)
	require.ErrorContains(t, m.Rename(ctx, "lio2025seifs-final", "occupied"), "destination directory is occupied")
	require.DirExists(t, filepath.Join(m.Root, "public/tasks/lio2025seifs-final"))
	_, err = storage.Lock(m.Root)
	require.Error(t, err, "maintenance must not overlap another storage owner")
}

func TestRenameRecoversAcrossFilesystemBoundary(t *testing.T) {
	for _, moved := range []bool{false, true} {
		t.Run(map[bool]string{false: "before move", true: "after move"}[moved], func(t *testing.T) {
			m, pool, store := setup(t)
			ctx := context.Background()
			seedTask(t, pool, "old")
			_, err := store.Upload([]byte("keep"), storage.Archive("old", "x"), "text/plain")
			require.NoError(t, err)
			_, err = pool.Exec(ctx, `INSERT INTO task_rename_pending(old_id,new_id) VALUES('old','new')`)
			require.NoError(t, err)
			if moved {
				require.NoError(t, os.Rename(filepath.Join(m.Root, "public/tasks/old"), filepath.Join(m.Root, "public/tasks/new")))
			}
			require.NoError(t, m.Prepare(ctx))
			require.NoError(t, m.Prepare(ctx))
			got, err := store.Download(storage.Archive("new", "x"))
			require.NoError(t, err)
			require.Equal(t, "keep", string(got))
			var pending int
			require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM task_rename_pending`).Scan(&pending))
			require.Zero(t, pending)
		})
	}
}

func TestMigrationVerifiesBytesAndCleanupPreservesSources(t *testing.T) {
	m, pool, store := setup(t)
	ctx := context.Background()
	seedTask(t, pool, "seifs")
	image, archive := []byte("image original"), []byte("pdf original\r\n")
	_, err := pool.Exec(ctx, `INSERT INTO task_images(task_short_id,file_name,width_px,height_px,filesize_bytes) VALUES('seifs','1.png',10,20,$1)`, len(image))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO task_archive_files(task_short_id,path,filesize_bytes) VALUES('seifs','original.pdf',$1)`, len(archive))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO task_storage_migration VALUES('seifs','image','1.png','seifs/old.png'),('seifs','archive','original.pdf','archive/seifs/original.pdf')`)
	require.NoError(t, err)
	// Missing input stops startup; nothing is silently omitted.
	require.Error(t, m.Prepare(ctx))
	_, err = store.Upload(image, "md-images/seifs/old.png", "image/png")
	require.NoError(t, err)
	_, err = store.Upload(archive, "archive/seifs/original.pdf", "application/pdf")
	require.NoError(t, err)
	_, err = store.Upload([]byte("unknown"), "orphan.bin", "application/octet-stream")
	require.NoError(t, err)
	require.NoError(t, m.Prepare(ctx))
	require.NoError(t, m.Prepare(ctx))
	task, err := repo.NewTaskPgRepo(pool).GetTask(ctx, "seifs")
	require.NoError(t, err)
	require.Equal(t, storage.Digest(image), task.MdImages[0].SHA256)
	got, err := store.Download(task.MdImages[0].ObjectKey)
	require.NoError(t, err)
	require.Equal(t, image, got)
	stale, err := m.Cleanup(ctx, false)
	require.NoError(t, err)
	require.Len(t, stale, 3)
	require.FileExists(t, filepath.Join(m.Root, "public/orphan.bin"))
	_, err = m.Cleanup(ctx, true)
	require.NoError(t, err)
	remaining, err := m.Cleanup(ctx, false)
	require.NoError(t, err)
	require.Empty(t, remaining)
	backups, err := filepath.Glob(filepath.Join(m.Root, "recovery/*/public/md-images/seifs/old.png"))
	require.NoError(t, err)
	require.Len(t, backups, 1)
	got, err = os.ReadFile(backups[0])
	require.NoError(t, err)
	require.Equal(t, image, got)
	got, err = store.Download(task.Archive[0].ObjectKey)
	require.NoError(t, err)
	require.Equal(t, archive, got)
}

func TestWrongStorageAndConcurrentDatabaseOwnerAreRejected(t *testing.T) {
	m, pool, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, m.Prepare(ctx))
	wrong := storage.Maintenance{DB: pool, Root: t.TempDir()}
	require.ErrorContains(t, wrong.Prepare(ctx), "storage identity marker")
	unlock, err := storage.LockDatabase(ctx, pool)
	require.NoError(t, err)
	defer unlock()
	_, err = storage.LockDatabase(ctx, pool)
	require.ErrorContains(t, err, "database is in use")
}

func TestUpgradePopulatedLegacySchema(t *testing.T) {
	ctx := context.Background()
	// Exercise the actual SQL transition, including the legacy composite PK and
	// the temporary path queue, rather than only an already migrated database.
	source := filepath.Join(conf.FindProjectRoot(), "postgres/migrate")
	oldMigrations := t.TempDir()
	entries, err := os.ReadDir(source)
	require.NoError(t, err)
	for _, e := range entries {
		if e.IsDir() || len(e.Name()) < 3 || e.Name()[:3] >= "077" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(source, e.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(oldMigrations, e.Name()), data, 0600))
	}
	env := func(key, fallback string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return fallback
	}
	cfg := pgtestdb.Config{DriverName: "pgx", Host: env("TEST_PG_HOST", "postgres"), Port: env("TEST_PG_PORT", "5432"), User: env("TEST_PG_USER", "postgres"), Password: env("TEST_PG_PASSWORD", "pw"), Options: "sslmode=disable"}
	db := pgtestdb.Custom(t, cfg, golangmigrator.New(oldMigrations))
	pool, err := pgxpool.New(ctx, db.URL())
	require.NoError(t, err)
	defer pool.Close()
	_, err = pool.Exec(ctx, `INSERT INTO tasks(short_id,full_name_dict,orig_lang,readme,origin_olympiad,difficulty_rating,checker,interactor,mem_lim_megabytes,cpu_time_lim_secs,illustr_img_object_key,width_px,height_px,filesize_bytes) VALUES('legacy','{"lv":"Legacy"}','lv','','LIO',1,'','',64,1,'old.png',2,3,10);
INSERT INTO task_images(task_short_id,object_key,file_name,width_px,height_px,filesize_bytes) VALUES('legacy','legacy/old.png','1.png',2,3,10);
INSERT INTO task_archive_files(task_short_id,path,object_key,filesize_bytes) VALUES('legacy','original.txt','archive/legacy/original.txt',8);`)
	require.NoError(t, err)
	root := t.TempDir()
	store, err := filestore.NewStore(filepath.Join(root, "public"))
	require.NoError(t, err)
	for _, key := range []string{"illustrations/old.png", "md-images/legacy/old.png", "archive/legacy/original.txt"} {
		_, err := store.Upload([]byte(key), key, "application/octet-stream")
		require.NoError(t, err)
	}
	sql, err := os.ReadFile(filepath.Join(source, "077_task_storage.up.sql"))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(sql))
	require.NoError(t, err)
	m := storage.Maintenance{DB: pool, Root: root}
	require.NoError(t, m.Prepare(ctx))
	task, err := repo.NewTaskPgRepo(pool).GetTask(ctx, "legacy")
	require.NoError(t, err)
	require.NotNil(t, task.IllustrImg)
	for key, want := range map[string]string{task.IllustrImg.ObjectKey: "illustrations/old.png", task.MdImages[0].ObjectKey: "md-images/legacy/old.png", task.Archive[0].ObjectKey: "archive/legacy/original.txt"} {
		got, err := store.Download(key)
		require.NoError(t, err)
		require.Equal(t, want, string(got))
	}
	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM task_storage_migration`).Scan(&n))
	require.Zero(t, n)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_name IN ('tasks','task_images','task_archive_files') AND column_name LIKE '%object_key%'`).Scan(&n))
	require.Zero(t, n)
}

func TestRenameTaskZipRoundtrip(t *testing.T) {
	m, pool, public := setup(t)
	tests, err := filestore.NewStore(filepath.Join(m.Root, "testfiles"))
	require.NoError(t, err)
	service := srvc.NewTaskSrvc(repo.NewTaskPgRepo(pool), public, tests)
	ctx := context.Background()
	data, err := os.ReadFile("../srvc/testdata/lio2026cuska.zip")
	require.NoError(t, err)
	id, importErr := service.ImportTaskFromZip(ctx, data, "before")
	require.NoError(t, importErr)
	exported, exportErr := service.ExportTaskAsZip(ctx, id)
	require.NoError(t, exportErr)
	before, err := taskzip.Read(exported)
	require.NoError(t, err)
	require.NoError(t, m.Rename(ctx, id, "after"))
	exported, exportErr = service.ExportTaskAsZip(ctx, "before")
	require.NoError(t, exportErr)
	after, err := taskzip.Read(exported)
	require.NoError(t, err)
	require.Equal(t, "after", after.ID)
	require.Equal(t, before.Tests, after.Tests)
	require.Equal(t, before.Testing, after.Testing)
	require.Equal(t, before.Subtasks, after.Subtasks)
	require.Equal(t, before.TestGroups, after.TestGroups)
	require.Equal(t, before.Statements, after.Statements)
	require.Equal(t, before.StatementImages, after.StatementImages)
	require.Equal(t, before.Solutions, after.Solutions)
	require.Equal(t, before.Archive, after.Archive)
}
