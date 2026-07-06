UPDATE task_images
SET s3_key = new_images.s3_key
FROM (
    VALUES
        ('fleksis', 'fleksis.png', 'fleksis/acd2963c062f.png'),
        ('adapteri', '2.png', 'adapteri/52ca9368b878.png')
) AS new_images(task_short_id, file_name, s3_key)
WHERE task_images.task_short_id = new_images.task_short_id
  AND task_images.file_name = new_images.file_name;
