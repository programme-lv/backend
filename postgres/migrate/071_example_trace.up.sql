-- Interactive tasks store their example interaction as a single trace file
-- (alternating message blocks separated by lines containing exactly "---").
-- Simple and checker tasks keep using input/output.
ALTER TABLE task_examples
ADD COLUMN IF NOT EXISTS trace TEXT;
