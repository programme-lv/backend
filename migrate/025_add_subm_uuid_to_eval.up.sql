ALTER TABLE evaluations
    RENAME COLUMN submission_uuid TO subm_uuid;

UPDATE evaluations e
SET subm_uuid = s.uuid
FROM submissions s
WHERE s.curr_eval_uuid = e.uuid
AND e.subm_uuid IS NULL;
