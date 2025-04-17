-- Create new task_images table
CREATE TABLE IF NOT EXISTS task_images (
    task_short_id TEXT NOT NULL,
    s3_uri TEXT NOT NULL,
    file_name TEXT,
    width_px INTEGER,
    height_px INTEGER,
    PRIMARY KEY (task_short_id, s3_uri),
    CONSTRAINT fk_task_images_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

-- Migrate data from task_md_statement_images to task_images
-- Convert s3_url to s3_uri and extract filename from the URL
-- We need to join with task_md_statements to get the task_short_id
INSERT INTO task_images (task_short_id, s3_uri, file_name, width_px, height_px)
SELECT 
    tms.task_short_id,
    REPLACE(tmsi.s3_url, 'https://proglv-public.s3.eu-central-1.amazonaws.com/', 's3://proglv-public/') AS s3_uri,
    -- Extract the filename from the path (everything after the last /)
    SUBSTRING(tmsi.s3_url FROM '[^/]+$') AS file_name,
    tmsi.width_px,
    tmsi.height_px
FROM 
    task_md_statement_images tmsi
JOIN 
    task_md_statements tms ON tmsi.md_statement_id = tms.id; 