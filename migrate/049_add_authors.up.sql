-- Add authors list as JSONB array of strings
ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS authors jsonb;

-- Initialize nulls to empty array
UPDATE tasks SET authors = '[]'::jsonb WHERE authors IS NULL;

