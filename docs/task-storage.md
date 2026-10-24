# Task assets and rare ID changes

All paths below are relative to `FILE_STORAGE_ROOT`:

```text
assets/tasks/<task-id>/
  images/<sha256>/<filename>
  illustration/<sha256>.<extension>
archive/tasks/<task-id>/<relative-filename>
cache/images/<sha256(original-key)>/<variant>.webp
testfiles/<sha256>.zst
exec/...
recovery/<timestamp>/...
```

Public originals live under `assets/`; admin-only archives under `archive/`.
Derived WebP variants live under `cache/` and are disposable: they are never
backed up, exported or considered source. Cache keys include a hash of the
original key, so replacing an image selects a new object. Public URLs keep the
stable `<original>.<variant>.webp` suffix; the handler derives the cache object
on demand. Hashes in image URLs prevent stale immutable caches. Archives never
pass the public asset allowlist.
Tests stay in the private shared content-addressed store: executions and
historical evaluations refer to their hashes independently of task identity.

Application tables store filenames, image SHA-256/extension/dimensions and archive
sizes. `ObjectKey` fields in service DTOs are computed runtime paths, never saved.
`modules/task/storage/paths.go` defines the layout.

## Migration

Migration 077 moves legacy paths into the temporary `task_storage_migration`
work queue and drops object-key columns from application tables. Before accepting
requests or starting workers, `Maintenance.Prepare` copies original files into
the new layout, fsyncs, reads back and verifies the bytes, and then commits the
metadata and removes that work item. An interruption resumes at the remaining
items. Missing files or conflicting destination bytes stop startup.

Migration 078 moves `public/tasks/<id>/{images,illustration}` to
`assets/tasks/<id>/...`, moves `public/tasks/<id>/archive` to
`archive/tasks/<id>`, deletes old `.list/.view/.full.webp` derivatives, and
records the layout transition in `task_storage_layout_pending` so an interrupted
move resumes. The cache is regenerated on demand.

Original legacy files remain until cleanup. Any remaining original archive ZIP
is unpacked after conversion by the existing compatibility path. Conversion
changes neither judging nor `tests_version`. SQL-only rollback is deliberately rejected; use a
paired database/filestore backup to return to an older release.

## Offline operations

The server and local `prglv` maintenance commands hold an exclusive Linux
file lock and a PostgreSQL advisory lock for their entire lifetime. A UUID
marker binds the database to its storage root, rejecting accidental mismatches.
Run one backend against a storage root;
stop it before maintenance. This protects submissions, profiles, asset writes
and HTTP caches without a separate online maintenance mode.

```sh
prglv rename seifs lio2025seifs
prglv storage-cleanup          # inventory, after completing pending migrations
prglv storage-cleanup --apply  # move unreferenced assets/archive/legacy files to recovery/
```

These commands use `POSTGRES_*`, `POSTGRES_MIGRATIONS_PATH` and
`FILE_STORAGE_ROOT` on the backend machine, **not** the HTTP API/admin key.
They are included in the backend image. Their filesystem must be the same
mounted filesystem used by the backend. See the shared operational runbook
[`../../docs/github/task-storage.md`](../../docs/github/task-storage.md).

## Rename recovery

After checking IDs and destination availability, rename commits a
`task_rename_pending` intent. It moves `assets/tasks/<old>` and
`archive/tasks/<old>` on the same filesystem and fsyncs their parents. A DB
transaction then updates the task PK (all child FKs cascade), historical
submissions, revision and the readme normalization note, inserts the old-ID
alias, and removes the intent. Tests, limits, profiles, timestamps and
evaluations survive. Old cache entries become unreferenced by construction.

Startup rolls pending intents forward before serving traffic. Exactly one of
the old/new directories must exist; ambiguous states stop recovery. A failure
after the directory move never requires guessing whether the DB committed:
the intent is deleted in the same transaction as the rename.

Aliases point directly to the current task and cascade through repeated renames.
Reserved IDs cannot be imported as new tasks. API task/asset paths redirect
with 308 (including write methods); task-page lookups return the canonical ID.
Submission writes use that canonical ID, and exact submission filters accept
aliases. Restarting the backend discards all in-process caches. Local TaskZip
directories and their `task.toml` IDs must be renamed separately or re-exported.

## Cleanup and verification

Cleanup first checks all referenced originals exist and verifies image checksums
and archive sizes. It retains referenced files under `assets/` and `archive/`,
moves other files (including leftover legacy `public/`) into a timestamped
recovery directory, and removes empty directories. It never garbage-collects
private tests, execution artifacts or `cache/`. Recovery is outside the served
tree; retain it with the paired backup until migration is verified.

```sh
TEST_PG_HOST=postgres go test -tags=integration ./modules/task/... ./modules/subm/pgrepo/...
go test ./...
go vet ./...
```

Integration cases cover preserved history, asset bytes, alias filters, repeated
renames, occupied IDs, filesystem lock contention, both sides of the rename
crash boundary, resumable conversion and non-destructive cleanup.
