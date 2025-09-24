-- Support multilingual full names for tasks
-- 1) Add new JSONB column
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS full_name_dict jsonb;

-- 2) Migrate existing data from full_name -> full_name_dict under key 'lv'
UPDATE tasks
SET full_name_dict = jsonb_build_object('lv', full_name)
WHERE (full_name_dict IS NULL OR full_name_dict = '{}'::jsonb)
  AND full_name IS NOT NULL;

-- 3) Drop old column
ALTER TABLE tasks DROP COLUMN IF EXISTS full_name;
