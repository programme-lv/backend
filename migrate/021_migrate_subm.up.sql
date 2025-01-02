BEGIN;

INSERT INTO submissions (
    uuid,
    content,
    author_uuid,
    task_shortid,
    lang_shortid,
    curr_eval_uuid,
    created_at
)
SELECT 
    subm_uuid,
    content,
    author_uuid,
    task_id,
    prog_lang_id,
    NULL,
    created_at
FROM subm_bkp;


DROP TABLE IF EXISTS subm_bkp CASCADE;
COMMIT;
