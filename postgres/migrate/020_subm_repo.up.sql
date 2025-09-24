-- Begin the transaction to ensure atomicity
BEGIN;

-- 1. Enable the uuid-ossp extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. Create the submissions table
CREATE TABLE submissions (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    author_uuid UUID NOT NULL,
    task_shortid VARCHAR(50) NOT NULL,
    lang_shortid VARCHAR(20) NOT NULL,
    curr_eval_uuid UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_author
        FOREIGN KEY(author_uuid)
        REFERENCES users(uuid)
        ON DELETE CASCADE
);

-- 3. Create the evaluations table
CREATE TABLE evaluations (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_uuid UUID NOT NULL,
    stage VARCHAR(20) NOT NULL,
    score_unit VARCHAR(20) NOT NULL,
    checker VARCHAR(50),
    interactor VARCHAR(50),
    cpu_lim_ms INTEGER NOT NULL,
    mem_lim_kib INTEGER NOT NULL,
    error_type VARCHAR(20),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_submission
        FOREIGN KEY(submission_uuid)
        REFERENCES submissions(uuid)
        ON DELETE CASCADE,
    CONSTRAINT stage_check
        CHECK (stage IN ('waiting', 'compiling', 'testing', 'finished')),
    CONSTRAINT score_unit_check
        CHECK (score_unit IN ('test', 'group', 'subtask'))
);

-- 4. Alter the submissions table to add a foreign key for curr_eval_uuid
ALTER TABLE submissions
    ADD CONSTRAINT fk_curr_eval
        FOREIGN KEY(curr_eval_uuid)
        REFERENCES evaluations(uuid)
        ON DELETE SET NULL;

-- 5. Create the subtasks table
CREATE TABLE subtasks (
    id SERIAL PRIMARY KEY,
    evaluation_uuid UUID NOT NULL,
    points INTEGER NOT NULL,
    description TEXT NOT NULL,
    st_tests INTEGER[] NOT NULL,
    CONSTRAINT fk_evaluation_subtasks
        FOREIGN KEY(evaluation_uuid)
        REFERENCES evaluations(uuid)
        ON DELETE CASCADE
);

-- 6. Create the test_groups table
CREATE TABLE test_groups (
    id SERIAL PRIMARY KEY,
    evaluation_uuid UUID NOT NULL,
    points INTEGER NOT NULL,
    subtasks INTEGER[] NOT NULL,
    tg_tests INTEGER[] NOT NULL,
    CONSTRAINT fk_evaluation_test_groups
        FOREIGN KEY(evaluation_uuid)
        REFERENCES evaluations(uuid)
        ON DELETE CASCADE
);

-- 7. Create the tests table
CREATE TABLE tests (
    id SERIAL PRIMARY KEY,
    evaluation_uuid UUID NOT NULL,
    ac BOOLEAN NOT NULL DEFAULT FALSE,
    wa BOOLEAN NOT NULL DEFAULT FALSE,
    tle BOOLEAN NOT NULL DEFAULT FALSE,
    mle BOOLEAN NOT NULL DEFAULT FALSE,
    re BOOLEAN NOT NULL DEFAULT FALSE,
    ig BOOLEAN NOT NULL DEFAULT FALSE,
    reached BOOLEAN NOT NULL DEFAULT FALSE,
    finished BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_evaluation_tests
        FOREIGN KEY(evaluation_uuid)
        REFERENCES evaluations(uuid)
        ON DELETE CASCADE
);

-- 8. Create indexes on foreign key columns to improve query performance
CREATE INDEX idx_submissions_author_uuid ON submissions(author_uuid);
CREATE INDEX idx_submissions_curr_eval_uuid ON submissions(curr_eval_uuid);
CREATE INDEX idx_evaluations_submission_uuid ON evaluations(submission_uuid);
CREATE INDEX idx_subtasks_evaluation_uuid ON subtasks(evaluation_uuid);
CREATE INDEX idx_test_groups_evaluation_uuid ON test_groups(evaluation_uuid);
CREATE INDEX idx_tests_evaluation_uuid ON tests(evaluation_uuid);

-- Commit the transaction
COMMIT;
