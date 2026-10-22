ALTER TABLE eval_test_results
DROP COLUMN IF EXISTS score;

ALTER TABLE evaluations
DROP COLUMN IF EXISTS received_score_tenths,
DROP COLUMN IF EXISTS possible_score_tenths;
