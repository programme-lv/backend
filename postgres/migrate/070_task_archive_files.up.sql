CREATE TABLE task_archive_files (
    task_short_id TEXT NOT NULL REFERENCES tasks (short_id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    object_key TEXT NOT NULL,
    filesize_bytes INTEGER NOT NULL,
    PRIMARY KEY (task_short_id, path)
);
