-- Partial test percentages let interactive tasks award one-decimal scores.
ALTER TABLE eval_test_results
ADD COLUMN IF NOT EXISTS score INTEGER NOT NULL DEFAULT 0;

ALTER TABLE evaluations
ADD COLUMN IF NOT EXISTS received_score_tenths INTEGER,
ADD COLUMN IF NOT EXISTS possible_score_tenths INTEGER;

UPDATE evaluations
SET received_score_tenths = received_score * 10
WHERE received_score_tenths IS NULL;

UPDATE evaluations
SET possible_score_tenths = possible_score * 10
WHERE possible_score_tenths IS NULL;
