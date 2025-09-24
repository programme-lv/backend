-- Add back s3:// prefix to task_images.s3_key
UPDATE task_images 
SET s3_key = CONCAT('s3://proglv-public/', s3_key)
WHERE s3_key NOT LIKE 's3://%';

-- Rename s3_key column back to s3_uri in task_images
ALTER TABLE task_images RENAME COLUMN s3_key TO s3_uri;

-- Add back HTTPS prefix to tasks.illustr_img_s3_key
UPDATE tasks 
SET illustr_img_s3_key = CONCAT('https://proglv-public.s3.eu-central-1.amazonaws.com/', illustr_img_s3_key)
WHERE illustr_img_s3_key != '' AND illustr_img_s3_key NOT LIKE 'https://%'; 