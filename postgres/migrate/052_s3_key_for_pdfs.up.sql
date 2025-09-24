-- Migrate task_pdf_statements.object_url to s3_key
-- 1) Add new column s3_key
ALTER TABLE task_pdf_statements ADD COLUMN IF NOT EXISTS s3_key text;

-- 2) Populate s3_key by stripping known prefixes from object_url
UPDATE task_pdf_statements
SET s3_key = REPLACE(
  REPLACE(COALESCE(object_url, ''), 'https://proglv-public.s3.eu-central-1.amazonaws.com/', ''),
  's3://proglv-dev/',
  ''
)
WHERE (s3_key IS NULL OR s3_key = '') AND object_url IS NOT NULL AND object_url <> '';

-- 3) Drop old column
ALTER TABLE task_pdf_statements DROP COLUMN IF EXISTS object_url;

-- Optional: you may add a NOT NULL later if desired after data is fully migrated
