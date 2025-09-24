-- Drop the created_at column from evaluation_scoring_subtasks and evaluation_scoring_testgroups
ALTER TABLE evaluation_scoring_subtasks DROP COLUMN created_at;
ALTER TABLE evaluation_scoring_testgroups DROP COLUMN created_at;

-- remove default for testgroup_points, accepted, wrong, untested, subtask_points
ALTER TABLE evaluation_scoring_subtasks ALTER COLUMN subtask_points DROP DEFAULT;
ALTER TABLE evaluation_scoring_testgroups ALTER COLUMN testgroup_points DROP DEFAULT;
ALTER TABLE evaluation_scoring_subtasks ALTER COLUMN accepted DROP DEFAULT;
ALTER TABLE evaluation_scoring_testgroups ALTER COLUMN accepted DROP DEFAULT;
ALTER TABLE evaluation_scoring_subtasks ALTER COLUMN wrong DROP DEFAULT;
ALTER TABLE evaluation_scoring_testgroups ALTER COLUMN wrong DROP DEFAULT;
ALTER TABLE evaluation_scoring_subtasks ALTER COLUMN untested DROP DEFAULT;
ALTER TABLE evaluation_scoring_testgroups ALTER COLUMN untested DROP DEFAULT;

CREATE TABLE evaluation_scoring_testset (
    eval_uuid UUID REFERENCES evaluations(eval_uuid) ON DELETE CASCADE,
    accepted INTEGER NOT NULL,
    wrong INTEGER NOT NULL,
    untested INTEGER NOT NULL,
    PRIMARY KEY (eval_uuid)
);