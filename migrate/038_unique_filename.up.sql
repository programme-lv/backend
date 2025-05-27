ALTER TABLE task_images 
ADD CONSTRAINT unique_task_filename 
UNIQUE (task_short_id, file_name);
