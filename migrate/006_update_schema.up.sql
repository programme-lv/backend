-- Migration: 006_update_schema.up.sql

BEGIN;

-- 1. Create runtime_data table
CREATE TABLE IF NOT EXISTS runtime_data (
    id SERIAL PRIMARY KEY,
    eval_uuid UUID NOT NULL,
    test_id INTEGER NOT NULL,
    stdout TEXT,
    stderr TEXT,
    exit_code INTEGER,
    cpu_time_millis INTEGER,
    wall_time_millis INTEGER,
    memory_kibi_bytes INTEGER,
    ctx_switches_forced BIGINT,
    exit_signal BIGINT,
    isolate_status VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Add foreign key columns to evaluation_tests for checker and subm runtime data
ALTER TABLE evaluation_tests
    ADD COLUMN IF NOT EXISTS checker_runtime_id INTEGER,
    ADD COLUMN IF NOT EXISTS subm_runtime_id INTEGER;

-- Add foreign key constraints after ensuring the columns are added
ALTER TABLE evaluation_tests
    ADD CONSTRAINT fk_checker_runtime
        FOREIGN KEY (checker_runtime_id)
        REFERENCES runtime_data(id)
        ON DELETE SET NULL,
    ADD CONSTRAINT fk_subm_runtime
        FOREIGN KEY (subm_runtime_id)
        REFERENCES runtime_data(id)
        ON DELETE SET NULL;

-- 7. Remove original runtime columns from evaluation_tests
ALTER TABLE evaluation_tests
    DROP COLUMN IF EXISTS checker_stdout,
    DROP COLUMN IF EXISTS checker_stderr,
    DROP COLUMN IF EXISTS checker_exit_code,
    DROP COLUMN IF EXISTS checker_cpu_time_millis,
    DROP COLUMN IF EXISTS checker_wall_time_millis,
    DROP COLUMN IF EXISTS checker_memory_kibi_bytes,
    DROP COLUMN IF EXISTS checker_ctx_switches_forced,
    DROP COLUMN IF EXISTS checker_exit_signal,
    DROP COLUMN IF EXISTS checker_isolate_status,
    DROP COLUMN IF EXISTS subm_stdout,
    DROP COLUMN IF EXISTS subm_stderr,
    DROP COLUMN IF EXISTS subm_exit_code,
    DROP COLUMN IF EXISTS subm_cpu_time_millis,
    DROP COLUMN IF EXISTS subm_wall_time_millis,
    DROP COLUMN IF EXISTS subm_memory_kibi_bytes,
    DROP COLUMN IF EXISTS subm_ctx_switches_forced,
    DROP COLUMN IF EXISTS subm_exit_signal,
    DROP COLUMN IF EXISTS subm_isolate_status;

-- 8. Create testlib_checker table
CREATE TABLE IF NOT EXISTS testlib_checker (
    id SERIAL PRIMARY KEY,
    checker_code TEXT NOT NULL UNIQUE
);

-- 9. Populate testlib_checker with distinct checker codes from evaluations
INSERT INTO testlib_checker (checker_code)
SELECT DISTINCT testlib_checker_code
FROM evaluations
WHERE testlib_checker_code IS NOT NULL;

-- 10. Add checker_id column to evaluations
ALTER TABLE evaluations
    ADD COLUMN IF NOT EXISTS checker_id INTEGER;

-- 11. Update evaluations to set checker_id based on testlib_checker_code
UPDATE evaluations e
SET checker_id = tc.id
FROM testlib_checker tc
WHERE e.testlib_checker_code = tc.checker_code;

-- 12. Add foreign key constraint to evaluations.checker_id
ALTER TABLE evaluations
    ADD CONSTRAINT fk_checker_id
        FOREIGN KEY (checker_id)
        REFERENCES testlib_checker(id)
        ON DELETE SET NULL;

-- 13. Remove testlib_checker_code column from evaluations
ALTER TABLE evaluations
    DROP COLUMN IF EXISTS testlib_checker_code;

-- 14. Rename tables
ALTER TABLE evaluation_scoring_subtasks
    RENAME TO evaluation_subtasks;

ALTER TABLE evaluation_scoring_testgroups
    RENAME TO evaluation_testgroups;

ALTER TABLE evaluation_scoring_testset
    RENAME TO evaluation_testset;

-- 15. Add description column to evaluation_subtasks
ALTER TABLE evaluation_subtasks
    ADD COLUMN IF NOT EXISTS description TEXT;

COMMIT;
