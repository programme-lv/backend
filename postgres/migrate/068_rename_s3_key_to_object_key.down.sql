ALTER TABLE tasks
	RENAME COLUMN archive_object_key TO archive_s3_key;

ALTER TABLE tasks
	RENAME COLUMN illustr_img_object_key TO illustr_img_s3_key;

ALTER TABLE task_images
	RENAME COLUMN object_key TO s3_key;
