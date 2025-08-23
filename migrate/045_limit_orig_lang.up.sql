-- Constrain original language to max 3 characters
-- Convert existing values safely by truncating to 3 characters
ALTER TABLE tasks
ALTER COLUMN orig_lang TYPE varchar(3)
USING LEFT(COALESCE(orig_lang, ''), 3);

