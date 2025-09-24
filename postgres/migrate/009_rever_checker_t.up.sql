BEGIN;
ALTER TABLE evaluations DROP COLUMN testlib_checker_id;
ALTER TABLE evaluations ADD COLUMN testlib_checker_code text;
DROP TABLE testlib_checker;
COMMIT;