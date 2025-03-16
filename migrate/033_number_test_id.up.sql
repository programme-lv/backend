UPDATE eval_test_results
SET test_id = subquery.row_num
FROM (
    SELECT id, evaluation_uuid, 
           ROW_NUMBER() OVER (PARTITION BY evaluation_uuid ORDER BY id) as row_num
    FROM eval_test_results
) AS subquery
WHERE eval_test_results.id = subquery.id;

ALTER TABLE eval_test_results ALTER COLUMN test_id SET NOT NULL;