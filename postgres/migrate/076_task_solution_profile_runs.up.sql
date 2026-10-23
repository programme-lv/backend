-- The profile UI reports requested runs, including after a backend restart.
-- 075 was already applied in production without this column.
ALTER TABLE task_solution_profiles
ADD COLUMN IF NOT EXISTS requested_runs INTEGER NOT NULL DEFAULT 1;
