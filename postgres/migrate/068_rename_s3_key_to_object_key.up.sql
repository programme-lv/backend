ALTER TABLE task_images
	RENAME COLUMN s3_key TO object_key;

ALTER TABLE tasks
	RENAME COLUMN illustr_img_s3_key TO illustr_img_object_key;

ALTER TABLE tasks
	RENAME COLUMN archive_s3_key TO archive_object_key;
