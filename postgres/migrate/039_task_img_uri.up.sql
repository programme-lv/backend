ALTER TABLE tasks 
RENAME COLUMN illustr_img_url TO illustr_img_uri;

ALTER TABLE tasks 
ADD COLUMN width_px integer,
ADD COLUMN height_px integer,
ADD COLUMN filesize_bytes integer;
