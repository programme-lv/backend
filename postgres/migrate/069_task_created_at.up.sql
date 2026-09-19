-- First insert time. Existing rows get NOW() at migrate; historical publish
-- time is not recoverable.
ALTER TABLE tasks
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
