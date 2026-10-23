-- Track explicit official-test/scoring revisions without rewriting past evaluations.
ALTER TABLE tasks
ADD COLUMN IF NOT EXISTS tests_version INTEGER NOT NULL DEFAULT 1;
