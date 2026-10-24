# TaskZip compatibility

Task upload and export use TaskZip major version 1 as specified by the Rust
`programme-lv/taskzip` project.

The backend accepts flat ZIP files and ZIP files with one top-level directory
whose name matches the task ID.
Core uncompressed content is limited to 512 MiB to keep the in-memory upload
path bounded.
It imports simple, checker, and interactive tasks.
Interactive packages provide `interactor.cpp`; every official test needs an
input and an answer file, but interactive answers may be empty when the
interactor ignores `ans`.
Interactive examples are interaction traces stored as `examples/NNN.txt`
(alternating message blocks separated by lines containing exactly `---`),
with optional translated notes in `examples/NNN.md`; simple and checker tasks
keep input/output examples.
Contestant attachments are also rejected because the backend has no storage or
delivery model for them.
Origin divisions are preserved as an ordered array during import and export.

`testspec/` is an authoring input. It is accepted but ignored during import
and omitted during export.

Admin updates negotiate with `GET /tasks/{taskId}/manifest`, which returns
section hashes, `content_etag`, and `tests_version` without test payloads.
`PATCH /tasks/{taskId}` accepts a sparse TaskZip in multipart field `task_zip`
plus a JSON `manifest` field listing changed sections and the base versions.
Omitted sections keep their stored value, so unchanged tests and content are
not uploaded. `Content-Type` is otherwise the same TaskZip v1 format.

The update applies in place, preserving the task row, submissions, and
evaluations. Test, scoring, checker, interactor, or limit changes are rejected
unless `allow_test_changes` is set; accepting them bumps
`tasks.tests_version`. `ReadPatch` parses the sparse archive without requiring the omitted files;
retained solutions and tests are referenced by the stored task. The complete
merged task is validated before persistence. A stale `base_content_etag`,
`base_tests_version`, or `revision` returns `409 task_conflict`. The revision
check is enforced in the same transaction as the write. A solution score is
never inferred from the archive: omitted scores remain unverified.

`archive/` holds leftover files (original PDFs, pre-TaskZip trees).
Each file is stored under `archive/tasks/{taskId}/{path}` and listed on the admin
Arhīvs tab. Official files under `tests/` remain the source of truth; keep
them out of `archive/`.
A task may store at most 256 archive files, 64 MiB per file, and 128 MiB total.
Admin upload and TaskZip import reject the write
when either cap would be exceeded.

Existing individually archived files are copied byte-for-byte. Any remaining
legacy original ZIP is unpacked on startup into individual archive files by
the same compatibility path as before. Asset paths are derived rather than
stored in the database; see [task storage](task-storage.md).

Export writes stored `archive/` files back into the zip. If the task has a
list illustration, export also writes `archive/illustration{ext}` from the
original image unless that path already exists. Re-import does not set
`IllustrImg` from that file.

Export is otherwise still lossy where the database cannot represent TaskZip data:

- metadata topic categories are flattened on import and exported as `topics`;
- classification slugs outside the TaskZip vocabularies are dropped on import;
- solution scores and structured origin notes are omitted;
- non-numeric origin years are omitted.

To verify compatibility, run backend task package tests and check a produced ZIP
with:

```sh
taskzip check path/to/task.zip
```

## Task readme and archive fidelity

The task package's `readme.md` is also the deviation ledger for the archive.
Whenever an edit changes the task's meaning, difficulty, tests, scoring,
statement, checker/interactor, reference solutions, limits, or historical
metadata, update `readme.md` in the same update with the original value or
archived bytes, the replacement, the reason, and how to reconstruct the
original olympiad version.

Do not silently fix a weak test, an ambiguous statement, or an incorrect
reference solution. Preserve replaced or deleted original artifacts in the
task's `archive/` or an immutable source such as an SVN revision, and record a
checksum. Test/scoring/checker/interactor changes bump `tests_version`, so rerun
reference solutions and calibration before finalizing limits.

Canonical policy and template:
[`task-archive-fidelity.md`](https://github.com/programme-lv/docs/blob/main/task-archive-fidelity.md).
