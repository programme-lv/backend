-- Add column to record original language of the task
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS orig_lang text;

-- Backfill existing tasks to 'lv' (Latvian) as default
UPDATE tasks SET orig_lang = 'lv' WHERE orig_lang IS NULL OR orig_lang = '';

