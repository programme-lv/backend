-- Add organization and year fields for task origin
ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS origin_org text,
  ADD COLUMN IF NOT EXISTS origin_year varchar(9);

-- Ensure origin_org and origin_year are empty string by default when NULL
UPDATE tasks SET origin_org = '' WHERE origin_org IS NULL;
UPDATE tasks SET origin_year = '' WHERE origin_year IS NULL;

