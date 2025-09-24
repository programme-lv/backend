-- Remove HTTPS prefix from tasks.illustr_img_s3_key
UPDATE tasks 
SET illustr_img_s3_key = REPLACE(illustr_img_s3_key, 'https://proglv-public.s3.eu-central-1.amazonaws.com/', '')
WHERE illustr_img_s3_key LIKE 'https://proglv-public.s3.eu-central-1.amazonaws.com/%';

-- Rename s3_uri column to s3_key in task_images
ALTER TABLE task_images RENAME COLUMN s3_uri TO s3_key;

-- Remove s3:// prefix from task_images.s3_key
UPDATE task_images 
SET s3_key = REPLACE(s3_key, 's3://proglv-public/', '')
WHERE s3_key LIKE 's3://proglv-public/%'; 