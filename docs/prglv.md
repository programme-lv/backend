# prglv administration CLI

`prglv` is the admin CLI for the programme.lv backend. It talks to a running
server over the admin HTTP API.

```text
prglv package <task>     build a TaskZip .zip without uploading
prglv upload <task>      package a TaskZip directory or .zip and import it
prglv update <task>      update statement, examples, images, solutions, or metadata in place
prglv manifest <task-id> show the remote manifest used for sparse updates
prglv export <task-id>   download a task as a TaskZip .zip archive
prglv completions fish   print a shell completion script (fish, bash, zsh)
prglv version
prglv help
```

## Install

```bash
go install github.com/programme-lv/backend/cmd/prglv@latest
```

Or build from the repo root:

```bash
go build -o prglv ./cmd/prglv
```

## Configuration

`upload`, `update`, and `export` need the backend base URL and the
server-to-server admin key:

| Setting | Flag | Environment |
| --- | --- | --- |
| Backend base URL | `--api-url` | `PRGLV_API_URL`, then `API_PUBLIC_BASE_URL`, default `http://localhost:8080` |
| Admin API key | `--api-key` | `PRGLV_API_KEY`, then `ADMIN_API_KEY` |

Flags win over the environment. The key is sent as
`Authorization: Bearer <key>` and is never logged.

## Package

```bash
prglv package path/to/task
prglv package path/to/task --out /tmp/task.zip
prglv package path/to/task --out - > task.zip
prglv package path/to/task --force
prglv package path/to/task --partial
```

`package` validates the TaskZip with the same reader as the backend and writes
the archive without using the network. The default output is `<task-id>.zip`;
an existing file is not overwritten without `--force`.

`--partial` reads the package with the sparse-update reader, which allows an
export without `tests/`. Use it only for an existing task and together with
`update --tests-unchanged`.

## Upload

```bash
prglv upload path/to/task
prglv upload path/to/task.zip
prglv upload path/to/task --override-id task-copy
prglv upload path/to/task --json
```

`<task>` can be a TaskZip directory or a `.zip` archive. The CLI validates the
package locally with the same reader the backend uses, so layout and metadata
errors surface before anything is sent. It then posts the archive to
`POST /tasks/upload` as the multipart field `task_zip`. `--override-id` sets
the `override_id` query parameter, which stores the task under a different ID
than the archive declares.

When `<task>` is a directory, the CLI builds a flat ZIP from its contents.
The archive may be flat or wrapped in a single directory named after the task
ID; both are accepted. The following top-level entries are never packaged
because the backend ignores them or rejects them outright:

- `testspec/` — accepted but ignored during import;
- `.taskzip/` — locally generated caches;
- `.git/`, `target/`, `__MACOSX/`, and `.DS_Store` — repository or OS noise.

`archive/` is packaged. The backend stores those leftover files individually
and shows them on the admin Arhīvs tab. Official `tests/` stay out of `archive/`.
Import rejects the zip if `archive/` exceeds 256 files, 64 MiB per file, or 128 MiB total.

Everything else is included, so a directory that already holds `task.toml`,
`tests/`, `statement/`, and `solutions/` at its root uploads as-is.

## Update

```bash
prglv update path/to/task
prglv update path/to/task --dry-run
prglv update path/to/task --allow-test-changes
prglv update path/to/task --tests-unchanged
prglv update path/to/task --yes --json
```

`update` negotiates before uploading. It packages the local TaskZip, fetches
`GET /tasks/{taskId}/manifest`, and compares section hashes for statements,
examples, solutions, readme, checker/interactor, and official tests. Only
changed sections plus `task.toml` are packed into the sparse patch sent to
`PATCH /tasks/{taskId}`. Unchanged test files are never uploaded; the task
manifest is small and contains no test payloads.

The command prints the plan, including added/modified/deleted file paths, and
asks for confirmation unless `--yes` is used. If tests or scoring change, the
confirmation requires typing the task ID instead of `y`. `--dry-run` prints the
plan and stops without sending anything. `--json` skips the prompt and prints
the result for automation.

The patch is applied in place: task ID, submissions, and past evaluations are
preserved. The multipart field `manifest` carries the base content ETag, base
`tests_version`, base task `revision`, the changed sections, and
`allow_test_changes`. A stale base returns `409 task_conflict`, so a concurrent
edit is not silently overwritten. The revision check is performed inside the
write transaction.

Official tests, scoring, execution limits, checker, and interactor are guarded:
if the patch changes any of them, the request fails with `409
task_tests_changed` unless `--allow-test-changes` is passed. That flag replaces
the test/scoring rows and increments the task's `tests_version`; previous
evaluations remain as historical snapshots. The `tests_version` and `revision`
are returned in the task view.

`--tests-unchanged` reads the package with the partial reader and assumes the
remote tests are still correct: it neither hashes nor sends `tests/`, and it
takes the remote test count and section hash from the manifest. This allows an
edit package produced by `export --no-tests` to update a task without
downloading 10–50 MB of tests. The flag requires an existing task and cannot
bring new tests; use a normal update with `--allow-test-changes` for that.
Changing only execution limits is better done through
`PATCH /tasks/{taskId}/limits`, which does not bump `tests_version` or make
solution profiles stale.

## Manifest

```bash
prglv manifest lio2026cuska
prglv manifest lio2026cuska --json
```

`manifest` fetches `GET /tasks/{taskId}/manifest`: the base revision,
`tests_version`, section digests, and the file hash map used to negotiate a
sparse update. It is also a cheap existence check; a missing task returns
`task_not_found` without downloading the archive.

## Export

```bash
prglv export lio2026cuska
prglv export lio2026cuska --out /tmp/task.zip
prglv export lio2026cuska --out - > task.zip
prglv export lio2026cuska --no-tests
prglv export lio2026cuska --force
```

The default output is `<task-id>.zip` in the current directory. Use `-` to
write to stdout. An existing file is not overwritten without `--force`.

`--no-tests` omits `tests/` and writes `<task-id>-no-tests.zip` by default. It
is a partial TaskZip: use it for inspection or as the input to
`update --tests-unchanged`, not for `upload`.

## Shell completions

Fish:

```fish
mkdir -p ~/.config/fish/completions
prglv completions fish > ~/.config/fish/completions/prglv.fish
```

Bash and zsh:

```bash
prglv completions bash > /etc/bash_completion.d/prglv
prglv completions zsh > "${fpath[1]}/_prglv"
```

The completion scripts are generated from the same command and flag
definitions as the help text, so they stay in sync.
