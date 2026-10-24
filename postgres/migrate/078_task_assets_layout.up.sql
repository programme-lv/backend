-- Move public task presentation assets to assets/tasks/<id>/, admin-only
-- archive files to archive/tasks/<id>/, and derived WebPs to cache/.
-- Paths are derived in code; this table only carries the durable migration
-- intent so startup can resume a directory move.
CREATE TABLE task_storage_layout_pending (
    layout TEXT PRIMARY KEY
);
INSERT INTO task_storage_layout_pending (layout) VALUES ('assets-v1');
