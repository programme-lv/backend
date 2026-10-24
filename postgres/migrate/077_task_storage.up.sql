-- Retain old locations only as a resumable migration work queue. Startup copies
-- and verifies bytes before removing these records. Normal application tables
-- contain asset metadata, not storage paths.
CREATE TABLE task_storage_identity (
    singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK(singleton),
    token UUID NOT NULL DEFAULT uuid_generate_v4(),
    attached BOOLEAN NOT NULL DEFAULT false
);
INSERT INTO task_storage_identity DEFAULT VALUES;

CREATE TABLE task_storage_migration (
    task_id TEXT NOT NULL REFERENCES tasks(short_id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    filename TEXT NOT NULL,
    old_key TEXT NOT NULL,
    PRIMARY KEY (task_id, kind, filename)
);
INSERT INTO task_storage_migration
SELECT short_id, 'illustration', '', illustr_img_object_key FROM tasks
WHERE COALESCE(illustr_img_object_key, '') <> '';
INSERT INTO task_storage_migration
SELECT task_short_id, 'image', file_name, object_key FROM task_images;
INSERT INTO task_storage_migration
SELECT task_short_id, 'archive', path, object_key FROM task_archive_files;
INSERT INTO task_storage_migration
SELECT short_id, 'zip', 'legacy-original.zip', archive_object_key FROM tasks
WHERE COALESCE(archive_object_key, '') <> '';

ALTER TABLE tasks DROP COLUMN illustr_img_object_key, DROP COLUMN archive_object_key,
    ADD COLUMN illustration_sha256 TEXT NOT NULL DEFAULT '',
    ADD COLUMN illustration_ext TEXT NOT NULL DEFAULT '';
ALTER TABLE task_images DROP CONSTRAINT task_images_pkey, DROP CONSTRAINT unique_task_filename;
ALTER TABLE task_images DROP COLUMN object_key,
    ADD COLUMN sha256 TEXT NOT NULL DEFAULT '',
    ADD PRIMARY KEY (task_short_id, file_name);
ALTER TABLE task_archive_files DROP COLUMN object_key;
ALTER TABLE submissions ALTER COLUMN task_shortid TYPE VARCHAR(64);

CREATE TABLE task_aliases (
    alias TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(short_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Preserve every existing FK's deletion semantics, adding update propagation.
DO $$
DECLARE c RECORD;
BEGIN
    FOR c IN SELECT conrelid::regclass AS tbl, conname, pg_get_constraintdef(oid) AS def
             FROM pg_constraint WHERE contype = 'f' AND confrelid = 'tasks'::regclass
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', c.tbl, c.conname);
        EXECUTE format('ALTER TABLE %s ADD CONSTRAINT %I %s ON UPDATE CASCADE', c.tbl, c.conname, c.def);
    END LOOP;
END $$;

-- Old public identifiers cannot be reused, even by a concurrent import.
CREATE FUNCTION reserve_task_alias() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM task_aliases WHERE alias = NEW.short_id) THEN
        RAISE EXCEPTION 'task identifier is reserved by an alias' USING ERRCODE = '23505';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER reserve_task_alias BEFORE INSERT OR UPDATE OF short_id ON tasks
FOR EACH ROW EXECUTE FUNCTION reserve_task_alias();

-- The durable intent bridges a filesystem rename and the DB transaction.
CREATE TABLE task_rename_pending (
    singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK (singleton),
    old_id TEXT NOT NULL,
    new_id TEXT NOT NULL
);
