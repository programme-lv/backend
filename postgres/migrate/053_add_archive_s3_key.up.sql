-- Add archive S3 key for full task archive ZIPs
ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS archive_s3_key text NOT NULL DEFAULT '';

