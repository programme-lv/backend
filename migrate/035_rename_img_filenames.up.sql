-- Rename file_names in task_images to sequential numbers (1.png, 2.png, etc.) for each task

-- First create a temporary table to store the new file names
CREATE TEMPORARY TABLE temp_task_images AS
WITH numbered_images AS (
    SELECT 
        task_short_id,
        s3_uri,
        ROW_NUMBER() OVER (PARTITION BY task_short_id ORDER BY s3_uri) || '.png' AS new_file_name,
        file_name AS old_file_name,
        width_px,
        height_px
    FROM task_images
)
SELECT * FROM numbered_images;

-- Now update the actual task_images table with the new file names
UPDATE task_images
SET file_name = temp.new_file_name
FROM temp_task_images temp
WHERE task_images.task_short_id = temp.task_short_id
AND task_images.s3_uri = temp.s3_uri;

-- Drop the temporary table
DROP TABLE temp_task_images; 