ALTER TABLE tasks 
DROP COLUMN width_px,
DROP COLUMN height_px,
DROP COLUMN filesize_bytes;

ALTER TABLE tasks 
RENAME COLUMN illustr_img_uri TO illustr_img_url; 