ALTER TABLE task_origin_notes
  ADD COLUMN IF NOT EXISTS info_short text;
