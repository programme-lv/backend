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

`archive/` and `testspec/` are authoring and archival inputs.
They are accepted but ignored during import and omitted during export.
Official files under `tests/` remain the source of truth.

Export is intentionally lossy where the database cannot represent TaskZip data:

- metadata topic categories are flattened on import and exported as `topics`;
- classification slugs outside the TaskZip vocabularies are dropped on import;
- solution scores and structured origin notes are omitted;
- non-numeric origin years are omitted;
- illustration-only assets and ignored directories are omitted.

To verify compatibility, run backend task package tests and check a produced ZIP
with:

```sh
taskzip check path/to/task.zip
```
