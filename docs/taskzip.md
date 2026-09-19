# TaskZip compatibility

Task upload and export use TaskZip major version 1 as specified by the Rust
`programme-lv/taskzip` project.

The backend accepts flat ZIP files and ZIP files with one top-level directory
whose name matches the task ID.
Core uncompressed content is limited to 512 MiB to keep the in-memory upload
path bounded.
It imports simple and checker tasks.
Interactive tasks are rejected because submission evaluation does not support
them yet.
Contestant attachments are also rejected because the backend has no storage or
delivery model for them.
Origin divisions are preserved as an ordered array during import and export.

`testspec/` is an authoring input. It is accepted but ignored during import
and omitted during export.

`archive/` holds leftover files (original PDFs, pre-TaskZip trees).
Each file is stored under `archive/{taskId}/{path}` and listed on the admin
Arhīvs tab. Official files under `tests/` remain the source of truth; keep
them out of `archive/`.
A task may store at most 256 archive files, 64 MiB per file, and 128 MiB total.
Admin upload, TaskZip import, and leftover-zip migration all reject the write
when either cap would be exceeded.

On startup, leftover `tasks.archive_object_key` zips (legacy
`og-file-archives/*.zip`) are unpacked into `task_archive_files` and the
key is cleared.

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
