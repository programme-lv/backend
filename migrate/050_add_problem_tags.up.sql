-- Add problem tags as JSONB array of strings
ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS problem_tags jsonb;

-- Initialize to empty array where NULL
UPDATE tasks SET problem_tags = '[]'::jsonb WHERE problem_tags IS NULL;

