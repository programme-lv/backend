ALTER TABLE evaluations
DROP CONSTRAINT fk_submission,
DROP COLUMN submission_uuid;
