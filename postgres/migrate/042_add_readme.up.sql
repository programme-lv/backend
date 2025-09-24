-- Add readme column to tasks to store markdown/notes
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS readme text;
