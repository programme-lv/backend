ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS origin_divisions jsonb NOT NULL DEFAULT '[]'::jsonb;
