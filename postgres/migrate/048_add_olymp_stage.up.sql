-- Add olympiad stage field
ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS olymp_stage varchar(20);

-- Backfill empty string for NULLs
UPDATE tasks SET olymp_stage = '' WHERE olymp_stage IS NULL;

